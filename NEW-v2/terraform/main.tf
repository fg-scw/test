terraform {
  required_version = ">= 1.6.0"

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "~> 2.62"
    }
  }
}

provider "scaleway" {
  region = var.region
  zone   = var.zone
}

locals {
  common_tags = concat(
    [
      "cluster=${var.cluster_name}",
      "environment=${var.environment}",
      "talos-version=${var.talos_version}",
      "managed-by=terraform",
    ],
    var.additional_tags
  )
}
