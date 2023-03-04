resource "azurerm_storage_account" "vault" {
  name                             = "${var.prefix}vault${var.suffix}"
  resource_group_name              = azurerm_resource_group.data.name
  location                         = azurerm_resource_group.data.location
  account_tier                     = "Standard"
  account_replication_type         = "ZRS"
  allow_nested_items_to_be_public  = false
  cross_tenant_replication_enabled = false
}

resource "azurerm_storage_container" "data" {
  name                  = "vault"
  storage_account_name  = azurerm_storage_account.vault.name
  container_access_type = "private"
}

resource "azurerm_role_assignment" "vault_data" {
  scope                = azurerm_resource_group.data.id
  role_definition_name = "Storage Account Contributor"
  principal_id         = azurerm_linux_function_app.server.identity.0.principal_id
}

resource "azurerm_storage_account" "server" {
  name                             = "${var.prefix}vaultfn${var.suffix}"
  resource_group_name              = azurerm_resource_group.server.name
  location                         = azurerm_resource_group.server.location
  account_tier                     = "Standard"
  account_replication_type         = "ZRS"
  allow_nested_items_to_be_public  = false
  cross_tenant_replication_enabled = false
}

resource "azurerm_storage_container" "server" {
  name                  = "package"
  storage_account_name  = azurerm_storage_account.server.name
  container_access_type = "private"
}
