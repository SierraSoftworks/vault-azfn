
resource "azurerm_storage_account" "server" {
  name                     = "${var.prefix}vaultfn${var.suffix}"
  resource_group_name      = azurerm_resource_group.server.name
  location                 = azurerm_resource_group.server.location
  account_tier             = "Standard"
  account_replication_type = "ZRS"
}

resource "azurerm_storage_container" "server" {
  name                  = "package"
  storage_account_name  = azurerm_storage_account.server.name
  container_access_type = "private"
}

data "azurerm_storage_account_blob_container_sas" "package" {
  connection_string = azurerm_storage_account.server.primary_connection_string
  container_name    = azurerm_storage_container.server.name

  start = timestamp()
  expiry = timeadd(timestamp(), "8000h")

  permissions {
    read   = true
    add    = false
    create = false
    write  = false
    delete = false
    list   = false
  }
}

resource "azurerm_app_service_plan" "server" {
  name                = "vault-serviceplan"
  location            = azurerm_resource_group.server.location
  resource_group_name = azurerm_resource_group.server.name
  kind                = "functionapp"
  reserved            = true

  sku {
    tier = "Dynamic"
    size = "Y1"
  }
}

resource "azurerm_function_app" "server" {
  name                       = "${var.prefix}vault${var.suffix}"
  location                   = azurerm_resource_group.server.location
  resource_group_name        = azurerm_resource_group.server.name
  app_service_plan_id        = azurerm_app_service_plan.server.id
  storage_account_name       = azurerm_storage_account.server.name
  storage_account_access_key = azurerm_storage_account.server.primary_access_key
  version                    = "~4"
  os_type                    = "linux"

  https_only = true

  site_config {
    app_scale_limit = 1
  }
  
  identity {
    type = "SystemAssigned"
  }

  app_settings = {
    "WEBSITE_RUN_FROM_PACKAGE": "${azurerm_storage_blob.package.url}${data.azurerm_storage_account_blob_container_sas.package.sas}",
    "AZURE_TENANT_ID": data.azurerm_client_config.current.tenant_id,
    "AZURE_ACCOUNT_NAME": azurerm_storage_account.vault.name,
    "AZURE_BLOB_CONTAINER": azurerm_storage_container.data.name,
    "AZURE_STORAGE_KEY": azurerm_storage_account.vault.primary_access_key,
    "VAULT_AZUREKEYVAULT_VAULT_NAME": "${var.prefix}vault${var.suffix}"
    "VAULT_AZUREKEYVAULT_KEY_NAME": "vault-unseal",
    "VAULT_API_ADDR": "https://${azurerm_function_app.server.default_hostname}",
    "OTEL_EXPORTER_OTLP_ENDPOINT": var.opentelemetry.endpoint,
    "OTEL_EXPORTER_OTLP_HEADERS": var.opentelemetry.headers,
    "OTEL_SERVICE_NAME": var.opentelemetry.service_name,
  }
}
