data "cloudflare_zones" "dns" {
  account = {
    id = var.cloudflare_account_id
  }
  name = var.domain
}

resource "cloudflare_dns_record" "vault" {
  zone_id = data.cloudflare_zones.dns.result[0].id
  name    = "vault"
  ttl     = 300
  type    = "CNAME"
  content = azurerm_function_app_flex_consumption.server.default_hostname
  proxied = false
}

resource "cloudflare_dns_record" "vault_asuid" {
  zone_id = data.cloudflare_zones.dns.result[0].id
  name    = "asuid.vault"
  ttl     = 300
  type    = "TXT"
  content = azurerm_function_app_flex_consumption.server.custom_domain_verification_id
  proxied = false
}
