resource "azurerm_service_plan" "server" {
  name                = "vault-serviceplan"
  location            = azurerm_resource_group.server.location
  resource_group_name = azurerm_resource_group.server.name
  os_type             = "Linux"
  sku_name            = "Y1"
}

data "azurerm_linux_function_app" "server" {
  name                = "${var.prefix}vault${var.suffix}"
  resource_group_name = azurerm_resource_group.server.name
}

resource "azurerm_linux_function_app" "server" {
  name                          = "${var.prefix}vault${var.suffix}"
  location                      = azurerm_resource_group.server.location
  resource_group_name           = azurerm_resource_group.server.name
  service_plan_id               = azurerm_service_plan.server.id
  storage_account_name          = azurerm_storage_account.server.name
  storage_uses_managed_identity = true

  https_only = true

  site_config {
    app_scale_limit = 1
    http2_enabled   = true

    cors {
      allowed_origins     = ["https://vault.${var.domain}"]
      support_credentials = false
    }
  }

  identity {
    type = "SystemAssigned"
  }

  lifecycle {
    ignore_changes = [
      site_config[0].application_insights_key,
      site_config[0].application_insights_connection_string,
    ]
  }

  app_settings = merge(
    data.azurerm_linux_function_app.server.app_settings,
    {
      "FUNCTIONS_WORKER_RUNTIME" : "custom",
      "WEBSITE_RUN_FROM_PACKAGE" : "${azurerm_storage_blob.package.url}${data.azurerm_storage_account_blob_container_sas.package.sas}",
      "AZURE_TENANT_ID" : data.azurerm_client_config.current.tenant_id,
      "AZURE_ACCOUNT_NAME" : azurerm_storage_account.vault.name,
      "AZURE_BLOB_CONTAINER" : azurerm_storage_container.data.name,
      "AZURE_STORAGE_KEY" : azurerm_storage_account.vault.primary_access_key,
      "VAULT_AZUREKEYVAULT_VAULT_NAME" : "${var.prefix}vault${var.suffix}"
      "VAULT_AZUREKEYVAULT_KEY_NAME" : "vault-unseal",
      "VAULT_API_ADDR" : "https://vault.${var.domain}",
      "OTEL_EXPORTER_OTLP_ENDPOINT" : "https://refinery.sierrasoftworks.com:443",
      "OTEL_EXPORTER_OTLP_HEADERS" : "x-honeycomb-team=${var.honeycomb_key}",
      "VAULT_AGENT_SET_EXECUTABLE_PATTERN" : "./plugins/*",
  })
}

resource "azurerm_app_service_custom_hostname_binding" "vault" {
  hostname            = "vault.${var.domain}"
  app_service_name    = azurerm_linux_function_app.server.name
  resource_group_name = azurerm_resource_group.server.name

  lifecycle {
    ignore_changes = [ssl_state, thumbprint]
  }

  depends_on = [
    azurerm_dns_txt_record.vault
  ]
}

resource "azurerm_app_service_managed_certificate" "vault" {
  custom_hostname_binding_id = azurerm_app_service_custom_hostname_binding.vault.id
}

resource "azurerm_app_service_certificate_binding" "vault" {
  hostname_binding_id = azurerm_app_service_custom_hostname_binding.vault.id
  certificate_id      = azurerm_app_service_managed_certificate.vault.id
  ssl_state           = "SniEnabled"
}
