data "cloudflare_zones" "dns" {
  filter = {
    account = {
      id = var.cloudflare_account_id
    }
    name = var.domain
  }
}

resource "azurerm_dns_cname_record" "vault" {
  name                = "vault"
  resource_group_name = "dns"
  zone_name           = var.domain
  ttl                 = 300
  record              = "${azurerm_linux_function_app.server.name}.azurewebsites.net"
}

resource "azurerm_dns_txt_record" "vault" {
  name                = "asuid.vault"
  resource_group_name = "dns"
  zone_name           = var.domain
  ttl                 = 300

  record {
    value = azurerm_linux_function_app.server.custom_domain_verification_id
  }
}

resource "cloudflare_dns_record" "vault" {
  zone_id = one(data.cloudflare_zones.dns.result).id
  name    = "vault"
  ttl     = 300
  type    = "CNAME"
  content = "${azurerm_linux_function_app.server.name}.azurewebsites.net"
  proxied = false
}

resource "cloudflare_dns_record" "vault_asuid" {
  zone_id = one(data.cloudflare_zones.dns.result).id
  name    = "asuid.vault"
  ttl     = 300
  type    = "TXT"
  content = azurerm_linux_function_app.server.custom_domain_verification_id
  proxied = false
}
