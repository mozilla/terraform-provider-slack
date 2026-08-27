// Copyright Mozilla Foundation
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

var _ datasource.DataSource = &UsersDataSource{}
var _ datasource.DataSourceWithConfigure = &UsersDataSource{}

type slackUsersClient interface {
	GetUsersContext(context.Context, ...slack.GetUsersOption) ([]slack.User, error)
}

// UsersDataSource lists the users visible to the configured Slack token.
type UsersDataSource struct {
	client slackUsersClient
}

type UsersDataSourceModel struct {
	TeamID         types.String     `tfsdk:"team_id"`
	IncludeDeleted types.Bool       `tfsdk:"include_deleted"`
	Users          []SlackUserModel `tfsdk:"users"`
}

type SlackUserModel struct {
	ID                types.String          `tfsdk:"id"`
	TeamID            types.String          `tfsdk:"team_id"`
	Name              types.String          `tfsdk:"name"`
	RealName          types.String          `tfsdk:"real_name"`
	Deleted           types.Bool            `tfsdk:"deleted"`
	Color             types.String          `tfsdk:"color"`
	TZ                types.String          `tfsdk:"tz"`
	TZLabel           types.String          `tfsdk:"tz_label"`
	TZOffset          types.Int64           `tfsdk:"tz_offset"`
	IsAdmin           types.Bool            `tfsdk:"is_admin"`
	IsOwner           types.Bool            `tfsdk:"is_owner"`
	IsPrimaryOwner    types.Bool            `tfsdk:"is_primary_owner"`
	IsRestricted      types.Bool            `tfsdk:"is_restricted"`
	IsUltraRestricted types.Bool            `tfsdk:"is_ultra_restricted"`
	IsBot             types.Bool            `tfsdk:"is_bot"`
	IsAppUser         types.Bool            `tfsdk:"is_app_user"`
	Updated           types.Int64           `tfsdk:"updated"`
	Locale            types.String          `tfsdk:"locale"`
	Profile           SlackUserProfileModel `tfsdk:"profile"`
}

type SlackUserProfileModel struct {
	FirstName             types.String `tfsdk:"first_name"`
	LastName              types.String `tfsdk:"last_name"`
	RealName              types.String `tfsdk:"real_name"`
	RealNameNormalized    types.String `tfsdk:"real_name_normalized"`
	DisplayName           types.String `tfsdk:"display_name"`
	DisplayNameNormalized types.String `tfsdk:"display_name_normalized"`
	Email                 types.String `tfsdk:"email"`
	Phone                 types.String `tfsdk:"phone"`
	Title                 types.String `tfsdk:"title"`
	StatusText            types.String `tfsdk:"status_text"`
	StatusEmoji           types.String `tfsdk:"status_emoji"`
	Image24               types.String `tfsdk:"image_24"`
	Image32               types.String `tfsdk:"image_32"`
	Image48               types.String `tfsdk:"image_48"`
	Image72               types.String `tfsdk:"image_72"`
	Image192              types.String `tfsdk:"image_192"`
	Image512              types.String `tfsdk:"image_512"`
}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all users visible to the configured token using Slack's `users.list` API.",
		Attributes: map[string]schema.Attribute{
			"include_deleted": schema.BoolAttribute{
				MarkdownDescription: "Whether deleted or deactivated users are included in `users`. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Workspace ID to list. Required when using an organization-level token and ignored by Slack for workspace-level tokens.",
				Optional:            true,
			},
			"users": schema.ListNestedAttribute{
				MarkdownDescription: "Users returned by Slack, including invited and deleted/deactivated users. Results are sorted by `id` for stable Terraform state.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: userSchemaAttributes(),
				},
			},
		},
	}
}

func userSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                  computedString("Slack user ID."),
		"team_id":             computedString("Slack workspace ID."),
		"name":                computedString("Username."),
		"real_name":           computedString("User's real name."),
		"deleted":             computedBool("Whether the user is deactivated."),
		"color":               computedString("User's display color."),
		"tz":                  computedString("IANA time zone name."),
		"tz_label":            computedString("Human-readable time zone label."),
		"tz_offset":           computedInt64("Time zone offset from UTC in seconds."),
		"is_admin":            computedBool("Whether the user is a workspace administrator."),
		"is_owner":            computedBool("Whether the user is a workspace owner."),
		"is_primary_owner":    computedBool("Whether the user is the workspace primary owner."),
		"is_restricted":       computedBool("Whether the user is a guest."),
		"is_ultra_restricted": computedBool("Whether the user is a single-channel guest."),
		"is_bot":              computedBool("Whether the user is a bot."),
		"is_app_user":         computedBool("Whether the user is an app user."),
		"updated":             computedInt64("Unix timestamp when the user was last updated."),
		"locale":              computedString("User's locale, when available."),
		"profile": schema.SingleNestedAttribute{
			MarkdownDescription: "Slack user profile. The token needs the `users:read.email` scope for `email` to be populated.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"first_name":              computedString("First name."),
				"last_name":               computedString("Last name."),
				"real_name":               computedString("Real name."),
				"real_name_normalized":    computedString("Normalized real name."),
				"display_name":            computedString("Display name."),
				"display_name_normalized": computedString("Normalized display name."),
				"email":                   computedString("Email address, when permitted by token scopes."),
				"phone":                   computedString("Phone number."),
				"title":                   computedString("Job title."),
				"status_text":             computedString("Status text."),
				"status_emoji":            computedString("Status emoji."),
				"image_24":                computedString("24 pixel avatar URL."),
				"image_32":                computedString("32 pixel avatar URL."),
				"image_48":                computedString("48 pixel avatar URL."),
				"image_72":                computedString("72 pixel avatar URL."),
				"image_192":               computedString("192 pixel avatar URL."),
				"image_512":               computedString("512 pixel avatar URL."),
			},
		},
	}
}

func computedString(description string) schema.StringAttribute {
	return schema.StringAttribute{MarkdownDescription: description, Computed: true}
}

func computedBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{MarkdownDescription: description, Computed: true}
}

func computedInt64(description string) schema.Int64Attribute {
	return schema.Int64Attribute{MarkdownDescription: description, Computed: true}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*slack.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *slack.Client, got %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Slack Client Not Configured", "Configure the Slack provider before reading users.")
		return
	}

	includeDeleted := true
	if !data.IncludeDeleted.IsNull() && !data.IncludeDeleted.IsUnknown() {
		includeDeleted = data.IncludeDeleted.ValueBool()
	}

	users, err := fetchUsers(ctx, d.client, data.TeamID.ValueString(), includeDeleted)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Slack Users", fmt.Sprintf("Slack users.list returned an error: %s", err))
		return
	}

	data.IncludeDeleted = types.BoolValue(includeDeleted)
	data.Users = users
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func fetchUsers(ctx context.Context, client slackUsersClient, teamID string, includeDeleted bool) ([]SlackUserModel, error) {
	options := []slack.GetUsersOption{slack.GetUsersOptionLimit(200)}
	if teamID != "" {
		options = append(options, slack.GetUsersOptionTeamID(teamID))
	}

	users, err := client.GetUsersContext(ctx, options...)
	if err != nil {
		return nil, err
	}

	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	result := make([]SlackUserModel, 0, len(users))
	for _, user := range users {
		if user.Deleted && !includeDeleted {
			continue
		}
		result = append(result, flattenSlackUser(user))
	}
	return result, nil
}

func flattenSlackUser(user slack.User) SlackUserModel {
	return SlackUserModel{
		ID:                types.StringValue(user.ID),
		TeamID:            types.StringValue(user.TeamID),
		Name:              types.StringValue(user.Name),
		RealName:          types.StringValue(user.RealName),
		Deleted:           types.BoolValue(user.Deleted),
		Color:             types.StringValue(user.Color),
		TZ:                types.StringValue(user.TZ),
		TZLabel:           types.StringValue(user.TZLabel),
		TZOffset:          types.Int64Value(int64(user.TZOffset)),
		IsAdmin:           types.BoolValue(user.IsAdmin),
		IsOwner:           types.BoolValue(user.IsOwner),
		IsPrimaryOwner:    types.BoolValue(user.IsPrimaryOwner),
		IsRestricted:      types.BoolValue(user.IsRestricted),
		IsUltraRestricted: types.BoolValue(user.IsUltraRestricted),
		IsBot:             types.BoolValue(user.IsBot),
		IsAppUser:         types.BoolValue(user.IsAppUser),
		Updated:           types.Int64Value(int64(user.Updated)),
		Locale:            types.StringValue(user.Locale),
		Profile: SlackUserProfileModel{
			FirstName:             types.StringValue(user.Profile.FirstName),
			LastName:              types.StringValue(user.Profile.LastName),
			RealName:              types.StringValue(user.Profile.RealName),
			RealNameNormalized:    types.StringValue(user.Profile.RealNameNormalized),
			DisplayName:           types.StringValue(user.Profile.DisplayName),
			DisplayNameNormalized: types.StringValue(user.Profile.DisplayNameNormalized),
			Email:                 types.StringValue(user.Profile.Email),
			Phone:                 types.StringValue(user.Profile.Phone),
			Title:                 types.StringValue(user.Profile.Title),
			StatusText:            types.StringValue(user.Profile.StatusText),
			StatusEmoji:           types.StringValue(user.Profile.StatusEmoji),
			Image24:               types.StringValue(user.Profile.Image24),
			Image32:               types.StringValue(user.Profile.Image32),
			Image48:               types.StringValue(user.Profile.Image48),
			Image72:               types.StringValue(user.Profile.Image72),
			Image192:              types.StringValue(user.Profile.Image192),
			Image512:              types.StringValue(user.Profile.Image512),
		},
	}
}
