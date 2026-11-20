resource "azurerm_service_plan" "server" {
  name                = "vault-flex-serviceplan"
  location            = azurerm_resource_group.server.location
  resource_group_name = azurerm_resource_group.server.name
  os_type             = "Linux"
  sku_name            = "FC1"
}

resource "azurerm_function_app_flex_consumption" "server" {
  # Flex apps can't be converted in place, so this is a new app alongside the retired Y1 one.
  name                = "${var.prefix}vault${var.suffix}-flex"
  location            = azurerm_resource_group.server.location
  resource_group_name = azurerm_resource_group.server.name
  service_plan_id     = azurerm_service_plan.server.id

  storage_container_type      = "blobContainer"
  storage_container_endpoint  = "${azurerm_storage_account.server.primary_blob_endpoint}${azurerm_storage_container.server.name}"
  storage_authentication_type = "SystemAssignedIdentity"

  runtime_name    = "custom"
  runtime_version = "1.0"

  # Vault's Azure storage backend has no HA locking, so never run more than one instance.
  maximum_instance_count = 1
  instance_memory_in_mb  = 2048

  dynamic "always_ready" {
    for_each = var.always_ready_instances > 0 ? [1] : []
    content {
      name           = "http"
      instance_count = var.always_ready_instances
    }
  }

  zip_deploy_file = data.archive_file.server.output_path

  https_only = true

  site_config {
    http2_enabled = true

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

  app_settings = {
    # With SystemAssignedIdentity storage auth, azurerm sets AzureWebJobsStorage to a connection
    # string with an empty AccountKey, which stops the Functions host from starting. Blank it and
    # point the host at its storage account by identity instead.
    "AzureWebJobsStorage" : "",
    "AzureWebJobsStorage__accountName" : azurerm_storage_account.server.name,
    "AzureWebJobsStorage__credential" : "managedidentity",
    "AZURE_TENANT_ID" : data.azurerm_client_config.current.tenant_id,
    "AZURE_ACCOUNT_NAME" : azurerm_storage_account.vault.name,
    "AZURE_BLOB_CONTAINER" : azurerm_storage_container.data.name,
    "AZURE_STORAGE_KEY" : azurerm_storage_account.vault.primary_access_key,
    "VAULT_AZUREKEYVAULT_VAULT_NAME" : "${var.prefix}vault${var.suffix}"
    "VAULT_AZUREKEYVAULT_KEY_NAME" : "vault-unseal",
    "VAULT_API_ADDR" : "https://vault.${var.domain}",
    "OTEL_EXPORTER_OTLP_ENDPOINT" : "https://telemetry.sierrasoftworks.com",
    "VAULT_AGENT_SET_EXECUTABLE_PATTERN" : "./plugins/*",
  }
}

resource "azurerm_app_service_custom_hostname_binding" "vault" {
  hostname            = "vault.${var.domain}"
  app_service_name    = azurerm_function_app_flex_consumption.server.name
  resource_group_name = azurerm_resource_group.server.name

  lifecycle {
    ignore_changes = [ssl_state, thumbprint]
  }

  depends_on = [
    azurerm_dns_txt_record.vault,
    azurerm_dns_cname_record.vault,
    cloudflare_dns_record.vault,
  ]
}

# Flex Consumption apps use site-scoped certificates (Microsoft.Web/sites/certificates), which
# azurerm's managed certificate and certificate binding resources don't support, so both the
# certificate and its SNI binding go through azapi.
resource "azapi_resource" "vault_certificate" {
  type      = "Microsoft.Web/sites/certificates@2024-11-01"
  name      = "vault.${var.domain}"
  parent_id = azurerm_function_app_flex_consumption.server.id
  location  = azurerm_resource_group.server.location

  body = {
    properties = {
      canonicalName = azurerm_app_service_custom_hostname_binding.vault.hostname
      serverFarmId  = azurerm_service_plan.server.id
    }
  }

  response_export_values = ["properties.thumbprint"]
}

resource "azapi_update_resource" "vault_certificate_binding" {
  type        = "Microsoft.Web/sites/hostNameBindings@2024-11-01"
  resource_id = azurerm_app_service_custom_hostname_binding.vault.id

  body = {
    properties = {
      sslState   = "SniEnabled"
      thumbprint = azapi_resource.vault_certificate.output.properties.thumbprint
    }
  }
}
