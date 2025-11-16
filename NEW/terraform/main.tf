# terraform {
#   required_version = ">= 1.9"

#   required_providers {
#     scaleway = {
#       source  = "scaleway/scaleway"
#       version = "~> 2.62"
#     }
#     http = {
#       source  = "hashicorp/http"
#       version = "~> 3.0"
#     }
#   }
# }

# provider "scaleway" {
#   access_key      = var.scw_access_key
#   secret_key      = var.scw_secret_key
#   project_id      = var.scw_project_id
#   region          = var.region
#   zone            = var.zone
# }

# # Récupérer l'IP publique de l'utilisateur
# data "http" "my_public_ip" {
#   url = "https://ifconfig.me/ip"
# }

# locals {
#   my_ip_cidr = "${chomp(data.http.my_public_ip.response_body)}/32"

#   common_tags = [
#     "cluster=${var.cluster_name}",
#     "environment=${var.environment}",
#     "managed-by=terraform",
#     "talos-version=${var.talos_version}"
#   ]
  
#   # Liste des zones pour multi-AZ
#   availability_zones = var.enable_multi_az ? [
#     "${var.region}-1",
#     "${var.region}-2",
#     "${var.region}-3"
#   ] : ["${var.region}-1"]
# }

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "~> 2.60"
    }
  }
}

# ---------------------------------------------------------------------------
# Provider Scaleway
#
# IMPORTANT :
# - On NE met PAS access_key / secret_key / project_id ici
#   pour éviter le warning "Multiple variable sources detected".
# - Utilise les variables d'environnement SCW_ACCESS_KEY, SCW_SECRET_KEY,
#   SCW_DEFAULT_PROJECT_ID, SCW_DEFAULT_REGION, SCW_DEFAULT_ZONE
#   ou ton ~/.config/scw/config.yaml
# ---------------------------------------------------------------------------

provider "scaleway" {
  region = var.region
  zone   = var.zone

  # Tags appliqués automatiquement côté provider sur certaines ressources
  # (pas utilisé partout, on a aussi local.common_tags)
  project_id = var.scw_project_id
}

# ---------------------------------------------------------------------------
# Locals globaux
# ---------------------------------------------------------------------------

locals {
  # Multi-AZ : si enable_multi_az = true, on utilise 3 AZ dans la région
  availability_zones = var.enable_multi_az ? [
    "${var.region}-1",
    "${var.region}-2",
    "${var.region}-3",
  ] : [
    var.zone
  ]

  # Tags communs pour toutes les ressources du cluster
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
