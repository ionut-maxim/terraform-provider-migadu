resource "migadu_rewrite" "example" {
  domain_name     = "example.com"
  name            = "catch-support"
  local_part_rule = "support-*"
  destinations    = ["alice@example.com"]
}

