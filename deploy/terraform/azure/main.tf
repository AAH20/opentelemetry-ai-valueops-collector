terraform {
  required_version = ">= 1.7.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}
}

variable "name" {
  description = "Short workload name used in Azure resource names."
  type    = string
  default = "aivalueops"
}

variable "location" {
  description = "Azure region for the platform foundation."
  type        = string
  default     = "westeurope"
}

resource "azurerm_resource_group" "this" {
  name     = "rg-${var.name}"
  location = var.location
}

resource "azurerm_log_analytics_workspace" "this" {
  name                = "log-${var.name}"
  location            = var.location
  resource_group_name = azurerm_resource_group.this.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_container_app_environment" "this" {
  name                       = "cae-${var.name}"
  location                   = var.location
  resource_group_name        = azurerm_resource_group.this.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.this.id
}

# Deliberately excludes the Container App until an immutable image digest and production rate card
# are supplied. This foundation creates no deployment that could accept unauthenticated telemetry.
output "container_app_environment_id" {
  description = "Foundation environment for a separately reviewed Container App deployment."
  value       = azurerm_container_app_environment.this.id
}
