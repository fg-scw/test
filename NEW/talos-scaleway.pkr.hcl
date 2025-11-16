packer {
  required_plugins {
    scaleway = {
      source  = "github.com/scaleway/scaleway"
      version = "~> 1.3"
    }
  }
}

# =========================
# Variables
# =========================

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
  type      = string
  sensitive = true
  default   = env("SCW_DEFAULT_PROJECT_ID")
}

variable "zone" {
  type    = string
  default = "fr-par-1"
}

variable "talos_version" {
  type    = string
  default = "v1.11.5"
}

variable "base_image" {
  description = "ID ou nom de l'image Scaleway de base (ex: ubuntu_jammy)"
  type        = string
  default     = "ubuntu_jammy"
}

variable "commercial_type" {
  description = "Type d’instance utilisé pendant le build"
  type        = string
  default     = "PRO2-XXS"
}

variable "volume_size" {
  description = "Taille du volume root (en GB) pour l’instance de build"
  type        = number
  default     = 10
}

variable "image_name" {
  description = "Nom optionnel de l’image Talos résultante"
  type        = string
  default     = ""
}

# =========================
# Locals
# =========================

locals {
  # On enlève les caractères non souhaités dans le timestamp
  timestamp       = regex_replace(timestamp(), "[- TZ:]", "")
  image_full_name = var.image_name != "" ? var.image_name : "talos-scaleway-${var.talos_version}-${local.timestamp}"
}

# =========================
# Source (builder Scaleway)
# =========================

source "scaleway" "talos" {
  access_key = var.scw_access_key
  secret_key = var.scw_secret_key
  project_id = var.scw_project_id
  zone       = var.zone

  # Image de base sur laquelle Packer va tourner le provisioning
  image           = var.base_image
  commercial_type = var.commercial_type

  image_name    = local.image_full_name
  snapshot_name = "talos-snapshot-${var.talos_version}-${local.timestamp}"

  # SSH
  communicator = "ssh"
  ssh_username = "root"

  # Volume root de l’instance temporaire
  root_volume {
    size_in_gb = var.volume_size
    type       = "sbs_volume"
  }

  # Tags de l’image
  tags = [
    "talos",
    "packer",
    var.talos_version,
  ]
}

# =========================
# Build
# =========================

build {
  name = "talos-scaleway-image"

  source "scaleway.talos" {
    name = "talos"
  }

  # S'assurer que cloud-init a terminé
  provisioner "shell" {
    inline = [
      "echo '==> Attente de cloud-init...'",
      "cloud-init status --wait || echo 'cloud-init status returned non-zero (ignore)'",
    ]
  }

  # Upload du script de build et du schematic Talos
  provisioner "file" {
    source      = "provision/build-image.sh"
    destination = "/tmp/build-image.sh"
  }

  provisioner "file" {
    source      = "provision/schematic.yaml"
    destination = "/tmp/schematic.yaml"
  }

  # Installation des dépendances + exécution du script de build
  provisioner "shell" {
    environment_vars = [
      "TALOS_VERSION=${var.talos_version}",
      # Ces variables peuvent être consommées par ton build-image.sh si tu veux
      "WORK_DIR=/tmp/talos-build",
      "SCHEMATIC_FILE=/tmp/schematic.yaml",
    ]

    # IMPORTANT :
    # - expect_disconnect: au cas où la connexion SSH saute quand le disque root est écrasé
    # - skip_clean: ne PAS essayer de supprimer le script temporaire après coup
    expect_disconnect = true
    skip_clean        = true

    inline = [
      "set -e",
      "echo '==> Installation des dépendances'",
      "export DEBIAN_FRONTEND=noninteractive",
      "apt-get update",
      "apt-get install -y curl wget jq zstd qemu-utils",
      "",
      "echo '==> Execution du script de build'",
      "chmod +x /tmp/build-image.sh",
      "bash /tmp/build-image.sh",
    ]
  }

  # Manifest de sortie
  post-processor "manifest" {
    output     = "manifest.json"
    strip_path = true
  }
}
