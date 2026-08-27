data "slack_users" "all" {}

output "slack_user_ids" {
  value = [for user in data.slack_users.all.users : user.id]
}
