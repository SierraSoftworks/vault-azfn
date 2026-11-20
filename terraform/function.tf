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

locals {
  # The SCM (Kudu) site lives at <app>.scm.<rest of the default hostname>.
  scm_hostname = replace(azurerm_function_app_flex_consumption.server.default_hostname, "/^([^.]+)\\./", "$1.scm.")
}

# Publishes the package to the app. This is deliberately not `zip_deploy_file` on the app itself:
# azurerm pushes that through the classic Kudu zipdeploy endpoints, which Flex apps answer with a
# 404, failing (and tainting) the app on every create. Publishing separately also lets the app's
# identity be granted access to the deployment container before the package is pushed.
resource "terraform_data" "deploy" {
  triggers_replace = [
    azurerm_function_app_flex_consumption.server.id,
    local.package_hash,
  ]

  provisioner "local-exec" {
    command = "${path.module}/scripts/deploy.sh"

    environment = {
      SCM_HOST    = local.scm_hostname
      PACKAGE     = data.archive_file.server.output_path
      DEPLOY_USER = nonsensitive(azurerm_function_app_flex_consumption.server.site_credential[0].name)
      # Marked nonsensitive so that Terraform doesn't suppress the script's output, which we need
      # to diagnose failed deployments. The script never prints it.
      DEPLOY_PASSWORD = nonsensitive(azurerm_function_app_flex_consumption.server.site_credential[0].password)
    }
  }

  depends_on = [
    azurerm_role_assignment.function_storage_blob_data_owner,
    azurerm_role_assignment.function_storage_account_contributor,
  ]
}

resource "azurerm_app_service_custom_hostname_binding" "vault" {
  hostname            = "vault.${var.domain}"
  app_service_name    = azurerm_function_app_flex_consumption.server.name
  resource_group_name = azurerm_resource_group.server.name

  lifecycle {
    ignore_changes = [ssl_state, thumbprint]
  }

  depends_on = [
    cloudflare_dns_record.vault,
    cloudflare_dns_record.vault_asuid,
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
