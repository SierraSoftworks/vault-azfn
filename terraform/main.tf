variable "location" {
  description = "The Azure location where this app will be deployed."
}

variable "prefix" {
  description = "The prefix used to generate globally unique names for resources."
}

variable "suffix" {
  description = "The suffix used to generate globally unique names for resources."
  default     = ""
}

variable "vault_version" {
  description = "The version of Hashicorp Vault to use."
  default     = "1.10.0"
}

variable "vault_agent_version" {
  description = "The version of the Vault agent to use."
  default     = "1.2.1"
}

variable "log_workspace_id" {
  description = "The workspace ID of the log analytics workspace to use."
}
