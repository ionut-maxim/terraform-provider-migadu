resource "migadu_forwarding" "example" {
  domain_name        = "example.com"
  mailbox_local_part = "alice"
  address            = "alice@personal.com"
}
