// We retrieve information on the client deploying this plan
// to determine tenant information.
data "azuread_client_config" "current" {}

variable "location" {
  description = "The Azure location where this app will be deployed."
  default     = "North Europe"
}

variable "domain" {
  description = "The domain root at which the Vault server will be accessible."
  default     = "sierrasoftworks.com"
}

variable "prefix" {
  description = "The prefix used to generate globally unique names for resources."
  default     = "sierra"
}

variable "suffix" {
  description = "The suffix used to generate globally unique names for resources."
  default     = ""
}

variable "vault_version" {
  description = "The version of Hashicorp Vault to use."
  default     = "1.15.1"
}

variable "vault_agent_version" {
  description = "The version of the Vault Azure Functions host agent to use."
  default     = "1.5.6"
}

variable "vault_github_plugin_version" {
  description = "The version of the Vault GitHub plugin to use."
  default     = "2.0.0"
}

variable "honeycomb_key" {
  description = "The Honeycomb API key to use for logging."
}
