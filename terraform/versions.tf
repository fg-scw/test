# Provider versions file
# This file is managed by Terraform and should be committed to version control

terraform {
  required_version = ">= 1.9"

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "~> 2.62"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.0"
    }
  }
}
