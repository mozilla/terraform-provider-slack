// Copyright Mozilla Foundation
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/slack-go/slack"
)

func TestFetchUsersPaginatesSortsAndMaps(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users.list" {
			t.Errorf("path = %q, want /users.list", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("limit"); got != "200" {
			t.Errorf("limit = %q, want 200", got)
		}
		if got := r.FormValue("team_id"); got != "T123" {
			t.Errorf("team_id = %q, want T123", got)
		}
		if got := r.FormValue("include_locale"); got != "true" {
			t.Errorf("include_locale = %q, want true", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch requests.Add(1) {
		case 1:
			if got := r.FormValue("cursor"); got != "" {
				t.Errorf("first cursor = %q, want empty", got)
			}
			fmt.Fprint(w, `{
  "ok": true,
  "members": [{
    "id": "U200", "team_id": "T123", "name": "second", "real_name": "Second User",
    "tz": "America/Toronto", "tz_label": "Eastern Daylight Time", "tz_offset": -14400,
    "is_admin": true, "updated": 1720000000, "locale": "en-CA",
    "profile": {"display_name": "Second", "email": "second@example.com", "image_48": "https://example.com/second.png"}
  }],
  "response_metadata": {"next_cursor": "next-page"}
}`)
		case 2:
			if got := r.FormValue("cursor"); got != "next-page" {
				t.Errorf("second cursor = %q, want next-page", got)
			}
			fmt.Fprint(w, `{
  "ok": true,
  "members": [{
    "id": "U100", "team_id": "T123", "name": "first", "deleted": true,
    "profile": {"display_name": "First"}
  }],
  "response_metadata": {"next_cursor": ""}
}`)
		default:
			t.Errorf("unexpected request %d", requests.Load())
		}
	}))
	defer server.Close()

	client := slack.New("test-token", slack.OptionAPIURL(server.URL+"/"), slack.OptionHTTPClient(server.Client()))
	users, err := fetchUsers(context.Background(), client, "T123")
	if err != nil {
		t.Fatalf("fetchUsers() error = %v", err)
	}

	if got := requests.Load(); got != 2 {
		t.Fatalf("request count = %d, want 2", got)
	}
	if got := []string{users[0].ID.ValueString(), users[1].ID.ValueString()}; !reflect.DeepEqual(got, []string{"U100", "U200"}) {
		t.Errorf("user IDs = %v, want sorted IDs", got)
	}
	if !users[0].Deleted.ValueBool() {
		t.Error("U100 deleted = false, want true")
	}
	if got := users[1].Profile.Email.ValueString(); got != "second@example.com" {
		t.Errorf("U200 email = %q, want second@example.com", got)
	}
	if got := users[1].TZOffset.ValueInt64(); got != -14400 {
		t.Errorf("U200 tz_offset = %d, want -14400", got)
	}
}

func TestFetchUsersReturnsSlackError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":false,"error":"missing_scope"}`)
	}))
	defer server.Close()

	client := slack.New("test-token", slack.OptionAPIURL(server.URL+"/"), slack.OptionHTTPClient(server.Client()))
	if _, err := fetchUsers(context.Background(), client, ""); err == nil || err.Error() != "missing_scope" {
		t.Fatalf("fetchUsers() error = %v, want missing_scope", err)
	}
}

func TestFlattenSlackUser(t *testing.T) {
	t.Parallel()

	user := flattenSlackUser(slack.User{
		ID:      "U123",
		IsBot:   true,
		Updated: slack.JSONTime(1234),
		Profile: slack.UserProfile{FirstName: "Ada", StatusEmoji: ":wave:"},
	})

	if user.ID.ValueString() != "U123" || !user.IsBot.ValueBool() || user.Updated.ValueInt64() != 1234 {
		t.Fatalf("flattenSlackUser() did not map top-level fields: %#v", user)
	}
	if user.Profile.FirstName.ValueString() != "Ada" || user.Profile.StatusEmoji.ValueString() != ":wave:" {
		t.Fatalf("flattenSlackUser() did not map profile fields: %#v", user.Profile)
	}
}
