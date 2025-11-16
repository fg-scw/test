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

provider "scaleway" {
  access_key      = var.scw_access_key
  secret_key      = var.scw_secret_key
  project_id      = var.scw_project_id
  region          = var.region
  zone            = var.zone
}

# Récupérer l'IP publique de l'utilisateur
data "http" "my_public_ip" {
  url = "https://ifconfig.me/ip"
}

locals {
  my_ip_cidr = "${chomp(data.http.my_public_ip.response_body)}/32"

  common_tags = [
    "cluster=${var.cluster_name}",
    "environment=${var.environment}",
    "managed-by=terraform",
    "talos-version=${var.talos_version}"
  ]
  
  # Liste des zones pour multi-AZ
  availability_zones = var.enable_multi_az ? [
    "${var.region}-1",
    "${var.region}-2",
    "${var.region}-3"
  ] : ["${var.region}-1"]
}
