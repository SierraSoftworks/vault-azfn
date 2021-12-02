resource "azurerm_application_insights" "vault" {
  name                = "app-vault"
  location            = azurerm_resource_group.server.location
  resource_group_name = azurerm_resource_group.server.name
  application_type    = "other"
  workspace_id       = var.log_workspace_id
}