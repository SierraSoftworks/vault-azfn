// We retrieve information on the client deploying this plan
// to determine tenant information.
data "azuread_client_config" "current" {}

variable "location" {
    description = "The Azure location where this app will be deployed."
}

variable "domain" {
    description = "The domain at which the Vault server will be accessible."
}

variable "prefix" {
    description = "The prefix used to generate globally unique names for resources."
}

variable "suffix" {
    description = "The suffix used to generate globally unique names for resources."
    default = ""
}

variable "vault_version" {
    description = "The version of Hashicorp Vault to use."
    default = "1.9.0"
}

variable "log_workspace_id" {
    description = "The workspace ID of the log analytics workspace to use."
}