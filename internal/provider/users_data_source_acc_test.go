// Copyright Mozilla Foundation
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if os.Getenv("SLACK_TOKEN") == "" {
				t.Fatal("SLACK_TOKEN must be set for acceptance tests")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "slack_users" "test" {}`,
				Check: resource.TestCheckResourceAttrWith("data.slack_users.test", "users.#", func(value string) error {
					count, err := strconv.Atoi(value)
					if err != nil {
						return fmt.Errorf("parse users count: %w", err)
					}
					if count == 0 {
						return fmt.Errorf("Slack users.list returned no users")
					}
					return nil
				}),
			},
		},
	})
}
