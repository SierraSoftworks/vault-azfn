data "azurerm_subscription" "primary" {
}

resource "azurerm_role_definition" "azure_auth" {
  name  = "vault-azure-auth"
  scope = data.azurerm_subscription.primary.id

  permissions {
    actions     = ["Microsoft.Compute/virtualMachines/*/read", "Microsoft.Compute/virtualMachineScaleSets/*/read"]
    not_actions = []
  }

  assignable_scopes = [
    data.azurerm_subscription.primary.id,
  ]
}

resource "azurerm_role_assignment" "azure_auth" {
  scope              = data.azurerm_subscription.primary.id
  principal_id       = azurerm_linux_function_app.server.identity.0.principal_id
  role_definition_id = azurerm_role_definition.azure_auth.role_definition_resource_id
}
