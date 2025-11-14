packer {
  required_plugins {
    scaleway = {
      source  = "github.com/scaleway/scaleway"
      version = "~> 1.3"
    }
  }
}

# Variables
variable "scw_access_key" {
  type      = string
  sensitive = true
  default   = env("SCW_ACCESS_KEY")
}

variable "scw_secret_key" {
  type      = string
  sensitive = true
  default   = env("SCW_SECRET_KEY")
}

variable "scw_project_id" {
  type    = string
  default = env("SCW_DEFAULT_PROJECT_ID")
}

variable "zone" {
  type    = string
  default = "fr-par-1"
}

variable "talos_version" {
  type    = string
  default = "v1.11.5"
}

variable "image_name" {
  type    = string
  default = "talos-scaleway"
}

variable "commercial_type" {
  type    = string
  default = "DEV1-S"  # Instance temporaire pour le build
}

variable "volume_size" {
  type    = number
  default = 20  # GB
}

# Source Ubuntu de base pour la construction
variable "base_image" {
  type    = string
  default = "ubuntu_jammy"  # Ubuntu 22.04 LTS
}

# Locals
locals {
  timestamp = formatdate("YYYYMMDDhhmmss", timestamp())
  image_full_name = "${var.image_name}-${var.talos_version}-${local.timestamp}"
  
  common_tags = [
    "talos",
    "kubernetes",
    var.talos_version,
    "managed-by-packer"
  ]
}

# Source Scaleway
source "scaleway" "talos" {
  project_id  = var.scw_project_id
  access_key  = var.scw_access_key
  secret_key  = var.scw_secret_key
  
  zone            = var.zone
  commercial_type = var.commercial_type
  image           = var.base_image
  
  # Noms des ressources
  server_name   = "packer-talos-builder-${local.timestamp}"
  image_name    = local.image_full_name
  snapshot_name = "talos-snapshot-${var.talos_version}-${local.timestamp}"
  
  # Configuration SSH
  ssh_username = "root"
  communicator = "ssh"
  
  # Volume de boot (utiliser l_ssd pour Packer, pas de SBS sur instances temporaires)
  root_volume {
    size_in_gb  = var.volume_size
    volume_type = "l_ssd"  # Local SSD pour instance temporaire Packer
  }
  
  # Tags
  tags = local.common_tags
  
  # Nettoyage
  remove_volume = true
}

# Build
build {
  name    = "talos-scaleway-image"
  sources = ["source.scaleway.talos"]
  
  # Attendre que cloud-init soit terminé
  provisioner "shell" {
    inline = [
      "echo 'Attente de cloud-init...'",
      "cloud-init status --wait || true",
      "sleep 5"
    ]
  }
  
  # Copier les fichiers nécessaires
  provisioner "file" {
    source      = "provision/build-image.sh"
    destination = "/tmp/build-image.sh"
  }
  
  provisioner "file" {
    source      = "provision/schematic.yaml"
    destination = "/tmp/schematic.yaml"
  }
  
  # Installer les dépendances et construire l'image Talos
  provisioner "shell" {
    environment_vars = [
      "TALOS_VERSION=${var.talos_version}"
    ]
    inline = [
      "set -e",
      "echo '==> Installation des dépendances'",
      "export DEBIAN_FRONTEND=noninteractive",
      "apt-get update",
      "apt-get install -y curl wget jq zstd qemu-utils",
      "",
      "echo '==> Exécution du script de build'",
      "chmod +x /tmp/build-image.sh",
      "bash /tmp/build-image.sh"
    ]
  }
  
  # Message de fin
  post-processor "manifest" {
    output     = "manifest.json"
    strip_path = true
  }
}
