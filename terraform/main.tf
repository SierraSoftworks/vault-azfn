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
  default     = "1.11.2"
}

variable "vault_agent_version" {
  description = "The version of the Vault agent to use."
  default     = "1.3.5"
}

variable "opentelemetry" {
  description = "The configuration used for the OpenTelemetry emission performed the Vault agent."
  default = {
    endpoint = ""
    service_name = "vault"
    headers = ""
  }
  type = object({
    endpoint = string
    service_name = string
    headers = string
  })
}
