terraform {
  required_version = ">= 1.0"
  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = ">= 2.63.0"
    }
  }
}

provider "scaleway" {
  zone   = "fr-par-1"
  region = "fr-par"
}

provider "scaleway" {
  alias  = "par2"
  zone   = "fr-par-2"
  region = "fr-par"
}

# provider "scaleway" {
#   alias  = "waw2"
#   zone   = "pl-waw-2"
#   region = "pl-waw"
# }

resource "scaleway_account_ssh_key" "cluster" {
  name       = "k0s-multi-az-cluster"
  public_key = var.ssh_public_key
}

# Ubuntu 22.04 images per zone
data "scaleway_instance_image" "ubuntu_par1" {
  architecture = "x86_64"
  name         = "Ubuntu 22.04 Jammy Jellyfish"
  zone         = "fr-par-1"
}

data "scaleway_instance_image" "ubuntu_par2" {
  provider     = scaleway.par2
  architecture = "x86_64"
  name         = "Ubuntu 22.04 Jammy Jellyfish"
  zone         = "fr-par-2"
}

# data "scaleway_instance_image" "ubuntu_waw2" {
#   provider     = scaleway.waw2
#   architecture = "x86_64"
#   name         = "Ubuntu 22.04 Jammy Jellyfish"
#   zone         = "pl-waw-2"
# }

# Security groups (wide open by default – tighten in production)
resource "scaleway_instance_security_group" "k0s_par1" {
  name                    = "k0s-par1"
  inbound_default_policy  = "accept"
  outbound_default_policy = "accept"
}

resource "scaleway_instance_security_group" "k0s_par2" {
  provider                = scaleway.par2
  name                    = "k0s-par2"
  inbound_default_policy  = "accept"
  outbound_default_policy = "accept"
}

# resource "scaleway_instance_security_group" "k0s_waw2" {
#   provider                = scaleway.waw2
#   name                    = "k0s-waw2"
#   inbound_default_policy  = "accept"
#   outbound_default_policy = "accept"
# }

# Public IPs
resource "scaleway_instance_ip" "controller_par1_ip" {
  zone = "fr-par-1"
}

resource "scaleway_instance_ip" "controller_par2_ip" {
  provider = scaleway.par2
  zone     = "fr-par-2"
}

# resource "scaleway_instance_ip" "controller_waw2_ip" {
#   provider = scaleway.waw2
#   zone     = "pl-waw-2"
# }

resource "scaleway_instance_ip" "gpu_par1_ip" {
  zone = "fr-par-1"
}

resource "scaleway_instance_ip" "gpu_par2_ip" {
  provider = scaleway.par2
  zone     = "fr-par-2"
}

# resource "scaleway_instance_ip" "gpu_waw2_ip" {
#   provider = scaleway.waw2
#   zone     = "pl-waw-2"
# }

# Controllers (HA control-plane)
resource "scaleway_instance_server" "controller_par1" {
  name  = "k0s-controller-fr-par-1"
  type  = var.controller_instance_type
  image = data.scaleway_instance_image.ubuntu_par1.id
  zone  = "fr-par-1"
  tags  = ["k0s", "controller", "fr-par-1"]

  ip_id             = scaleway_instance_ip.controller_par1_ip.id
  security_group_id = scaleway_instance_security_group.k0s_par1.id

}

resource "scaleway_instance_server" "controller_par2" {
  provider = scaleway.par2

  name  = "k0s-controller-fr-par-2"
  type  = var.controller_instance_type
  image = data.scaleway_instance_image.ubuntu_par2.id
  zone  = "fr-par-2"
  tags  = ["k0s", "controller", "fr-par-2"]

  ip_id             = scaleway_instance_ip.controller_par2_ip.id
  security_group_id = scaleway_instance_security_group.k0s_par2.id

}

# resource "scaleway_instance_server" "controller_waw2" {
#   provider = scaleway.waw2

#   name  = "k0s-controller-pl-waw-2"
#   type  = var.controller_instance_type
#   image = data.scaleway_instance_image.ubuntu_waw2.id
#   zone  = "pl-waw-2"
#   tags = ["k0s", "controller", "pl-waw-2"]

#   ip_id             = scaleway_instance_ip.controller_waw2_ip.id
#   security_group_id = scaleway_instance_security_group.k0s_waw2.id

#   root_volume {
#     size_in_gb  = 150
#     volume_type = "sbs_volume"
#   }

# }

# GPU workers (L4-1-24G)
# IMPORTANT: 150GB disk to prevent DiskPressure issues with GPU images
resource "scaleway_instance_server" "gpu_par1" {
  name  = "k0s-gpu-fr-par-1"
  type  = var.gpu_instance_type
  image = data.scaleway_instance_image.ubuntu_par1.id
  zone  = "fr-par-1"
  tags  = ["k0s", "gpu", "fr-par-1"]

  ip_id             = scaleway_instance_ip.gpu_par1_ip.id
  security_group_id = scaleway_instance_security_group.k0s_par1.id

  root_volume {
    size_in_gb  = 150
    volume_type = "sbs_volume"
  }
}

resource "scaleway_instance_server" "gpu_par2" {
  provider = scaleway.par2

  name  = "k0s-gpu-fr-par-2"
  type  = var.gpu_instance_type
  image = data.scaleway_instance_image.ubuntu_par2.id
  zone  = "fr-par-2"
  tags  = ["k0s", "gpu", "fr-par-2"]

  ip_id             = scaleway_instance_ip.gpu_par2_ip.id
  security_group_id = scaleway_instance_security_group.k0s_par2.id

  root_volume {
    size_in_gb  = 150
    volume_type = "sbs_volume"
  }
}

# resource "scaleway_instance_server" "gpu_waw2" {
#   provider = scaleway.waw2

#   name  = "k0s-gpu-pl-waw-2"
#   type  = var.gpu_instance_type
#   image = data.scaleway_instance_image.ubuntu_waw2.id
#   zone  = "pl-waw-2"
#   tags = ["k0s", "gpu", "pl-waw-2"]

#   ip_id             = scaleway_instance_ip.gpu_waw2_ip.id
#   security_group_id = scaleway_instance_security_group.k0s_waw2.id

#   root_volume {
#     size_in_gb  = 150
#     volume_type = "sbs_volume"
#   }
# }

# Outputs
output "controller_par1_ip" {
  value       = scaleway_instance_ip.controller_par1_ip.address
  description = "Public IP of controller in fr-par-1 (also used for Ray head & vLLM access by default)."
}

output "controller_par2_ip" {
  value       = scaleway_instance_ip.controller_par2_ip.address
  description = "Public IP of controller in fr-par-2."
}

# output "controller_waw2_ip" {
#   value       = scaleway_instance_ip.controller_waw2_ip.address
#   description = "Public IP of controller in pl-waw-2."
# }

output "gpu_par1_ip" {
  value       = scaleway_instance_ip.gpu_par1_ip.address
  description = "Public IP of GPU worker in fr-par-1."
}

output "gpu_par2_ip" {
  value       = scaleway_instance_ip.gpu_par2_ip.address
  description = "Public IP of GPU worker in fr-par-2."
}

# output "gpu_waw2_ip" {
#   value       = scaleway_instance_ip.gpu_waw2_ip.address
#   description = "Public IP of GPU worker in pl-waw-2."
# }

output "k0sctl_config" {
  value = templatefile("${path.module}/k0sctl.yaml.tpl", {
    controller_par1_ip = scaleway_instance_ip.controller_par1_ip.address
    controller_par2_ip = scaleway_instance_ip.controller_par2_ip.address
    #controller_waw2_ip = scaleway_instance_ip.controller_waw2_ip.address
    gpu_par1_ip = scaleway_instance_ip.gpu_par1_ip.address
    gpu_par2_ip = scaleway_instance_ip.gpu_par2_ip.address
    #gpu_waw2_ip        = scaleway_instance_ip.gpu_waw2_ip.address
    ssh_key_path = var.ssh_private_key_path
  })
  description = "Rendered k0sctl configuration for the multi-AZ GPU cluster."
  sensitive   = true
}
