resource "migadu_domain" "example" {
  name = "example.com"
}

# The domain exposes all required DNS records as computed attributes.
# Use them to configure your DNS provider — here with Cloudflare.

resource "cloudflare_record" "migadu_verify" {
  zone_id = var.cloudflare_zone_id
  name    = migadu_domain.example.dns_verification.name
  type    = upper(migadu_domain.example.dns_verification.type)
  content = migadu_domain.example.dns_verification.value
}

resource "cloudflare_record" "migadu_spf" {
  zone_id = var.cloudflare_zone_id
  name    = migadu_domain.example.spf.name
  type    = upper(migadu_domain.example.spf.type)
  content = migadu_domain.example.spf.value
}

resource "cloudflare_record" "migadu_dmarc" {
  zone_id = var.cloudflare_zone_id
  name    = migadu_domain.example.dmarc.name
  type    = upper(migadu_domain.example.dmarc.type)
  content = migadu_domain.example.dmarc.value
}

resource "cloudflare_record" "migadu_dkim" {
  for_each = { for r in migadu_domain.example.dkim_records : r.name => r }
  zone_id  = var.cloudflare_zone_id
  name     = each.value.name
  type     = upper(each.value.type)
  content  = each.value.value
}

resource "cloudflare_record" "migadu_mx" {
  for_each = { for r in migadu_domain.example.mx_records : r.value => r }
  zone_id  = var.cloudflare_zone_id
  name     = each.value.name
  type     = upper(each.value.type)
  content  = each.value.value
  priority = each.value.priority
}
