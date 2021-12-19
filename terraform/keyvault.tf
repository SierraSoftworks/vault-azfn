# The user which is currently running terraform
data "azurerm_client_config" "current" {}
data "azuread_client_config" "current" {}

resource "azurerm_key_vault" "unseal" {
  name                       = "${var.prefix}vault${var.suffix}"
  location                   = var.location
  resource_group_name        = azurerm_resource_group.server.name
  enable_rbac_authorization  = false
  tenant_id                  = data.azuread_client_config.current.tenant_id
  soft_delete_retention_days = 30

  sku_name = "standard"

  enabled_for_deployment = true

  access_policy {
    tenant_id = azurerm_function_app.server.identity.0.tenant_id
    object_id = azurerm_function_app.server.identity.0.principal_id
    key_permissions = [
      "get",
      "wrapKey",
      "unwrapKey",
    ]
  }

  network_acls {
    default_action = "Allow"
    bypass         = "AzureServices"
  }
}

resource "azurerm_key_vault_key" "unseal" {
  name         = "vault-unseal"
  key_vault_id = azurerm_key_vault.unseal.id
  key_type     = "RSA"
  key_size     = 2048

  key_opts = [
    "wrapKey",
    "unwrapKey",
  ]
}
