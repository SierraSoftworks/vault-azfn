terraform {
  required_version = ">= 1.1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.84.0"
    }

    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.46.0"
    }
  }
}

terraform {
  cloud {
    organization = "sierrasoftworks"

    workspaces {
      name = "vault"
    }
  }
}

provider "azurerm" {
  features {}

  // NOTE: You can retrieve this secret using `op read op://epfkgzb2bz4msye2xrhffiz3se/jrlwg64m56hkbkbfvgljfkwcfy/Azure/client_secret`
  subscription_id = var.azure_subscription
  tenant_id       = var.azure_tenant
  client_id       = var.azure_client_id
  client_secret   = var.azure_client_secret
}

provider "azuread" {
  // NOTE: You can retrieve this secret using `op read op://epfkgzb2bz4msye2xrhffiz3se/jrlwg64m56hkbkbfvgljfkwcfy/Azure/client_secret`
  tenant_id     = var.azure_tenant
  client_id     = var.azure_client_id
  client_secret = var.azure_client_secret
}

variable "azure_subscription" {

}

variable "azure_tenant" {

}

variable "azure_client_id" {
  sensitive = true
}

variable "azure_client_secret" {
  sensitive = true
}
