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

