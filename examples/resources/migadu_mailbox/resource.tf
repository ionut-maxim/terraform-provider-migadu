resource "migadu_mailbox" "example" {
  domain_name = "example.com"
  local_part  = "alice"
  name        = "Alice"
  password    = "changeme"
}
