terraform {
  required_version = ">= 0.15.0"

  required_providers {
    azuread = {
      source = "hashicorp/azuread"
      version = "1.6.0"
    }
  }
}