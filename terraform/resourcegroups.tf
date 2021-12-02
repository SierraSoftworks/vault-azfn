resource "azurerm_resource_group" "server" {
  name     = "app-vault"
  location = "${var.location}"
}

resource "azurerm_resource_group" "data" {
  name     = "app-vault-data"
  location = "${var.location}"
}