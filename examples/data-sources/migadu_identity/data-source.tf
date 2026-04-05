data "migadu_identity" "example" {
  domain_name        = "example.com"
  mailbox_local_part = "alice"
  local_part         = "alice.work"
}
