# ============================================================================
# Instance Bastion pour Bootstrap Talos
# ============================================================================

data "scaleway_instance_image" "ubuntu" {
  name         = "Ubuntu 22.04 Jammy Jellyfish"
  architecture = "x86_64"
  zone         = var.zone
  latest       = true
}

resource "scaleway_instance_server" "bastion" {
  count = var.enable_bootstrap_bastion ? 1 : 0

  name              = "${var.cluster_name}-bootstrap-bastion"
  type              = var.bastion_instance_type
  image             = data.scaleway_instance_image.ubuntu.id
  zone              = var.zone
  security_group_id = scaleway_instance_security_group.bastion[0].id

  root_volume {
    size_in_gb            = 20
    volume_type           = "sbs_volume"
    delete_on_termination = true
  }

  # IP publique pour accès SSH initial
  ip_id = scaleway_instance_ip.bastion[0].id

  # Attachement au Private Network pour accéder aux nœuds Talos
  private_network {
    pn_id = scaleway_vpc_private_network.kubernetes.id
  }

  # Cloud-init pour installer les outils et bootstrapper
  user_data = {
    cloud-init = templatefile("${path.module}/templates/bootstrap-cloud-init.yaml", {
      cluster_name       = var.cluster_name
      talos_version      = var.talos_version
      k8s_api_endpoint   = local.k8s_api_endpoint
      control_plane_ips  = join(" ", local.control_plane_ips)
      worker_ips         = join(" ", local.worker_ips)
      cilium_patch       = base64encode(file("${path.root}/../cilium-patch.yaml"))
    })
  }

  tags = concat(local.common_tags, [
    "role=bootstrap-bastion",
    "temporary=true"
  ])

  depends_on = [
    scaleway_vpc_gateway_network.kubernetes,
    scaleway_instance_server.control_plane,
    scaleway_instance_server.workers,
  ]
}

# IP publique pour le bastion
resource "scaleway_instance_ip" "bastion" {
  count = var.enable_bootstrap_bastion ? 1 : 0
  zone  = var.zone
  tags  = local.common_tags
}

# Security Group pour le bastion
resource "scaleway_instance_security_group" "bastion" {
  count = var.enable_bootstrap_bastion ? 1 : 0

  name                    = "${var.cluster_name}-bastion-sg"
  description             = "Security group for bootstrap bastion"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  stateful                = true
  zone                    = var.zone
  tags                    = local.common_tags

  # SSH depuis n'importe où (temporaire pour le bootstrap)
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 22
    ip_range = "0.0.0.0/0"
  }

  # ICMP
  inbound_rule {
    action   = "accept"
    protocol = "ICMP"
    ip_range = var.private_network_cidr
  }
}

# ============================================================================
# Outputs
# ============================================================================

output "bastion_ip" {
  description = "IP publique du bastion de bootstrap"
  value       = var.enable_bootstrap_bastion ? scaleway_instance_ip.bastion[0].address : null
}

output "bastion_ssh_connection" {
  description = "Commande SSH pour se connecter au bastion"
  value       = var.enable_bootstrap_bastion ? "ssh root@${scaleway_instance_ip.bastion[0].address}" : null
}