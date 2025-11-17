# ============================================================================
# VPC et Private Network avec IPAM
# ============================================================================

resource "scaleway_vpc" "kubernetes" {
  name = "${var.cluster_name}-vpc"
  tags = local.common_tags
}

resource "scaleway_vpc_private_network" "kubernetes" {
  name   = "${var.cluster_name}-pn"
  vpc_id = scaleway_vpc.kubernetes.id
  tags   = local.common_tags

  ipv4_subnet {
    subnet = var.private_network_cidr
  }
}

# ============================================================================
# Public Gateway pour NAT (accès internet sortant)
# ============================================================================

resource "scaleway_vpc_public_gateway_ip" "main" {
  zone = var.zone
  tags = local.common_tags
}

resource "scaleway_vpc_public_gateway" "main" {
  name            = "${var.cluster_name}-pgw"
  type            = var.public_gateway_type
  zone            = var.zone
  ip_id           = scaleway_vpc_public_gateway_ip.main.id
  bastion_enabled = var.enable_bastion_on_gateway
  bastion_port    = var.bastion_ssh_port
  tags            = local.common_tags
}

resource "scaleway_vpc_gateway_network" "kubernetes" {
  gateway_id         = scaleway_vpc_public_gateway.main.id
  private_network_id = scaleway_vpc_private_network.kubernetes.id
  enable_masquerade  = true

  ipam_config {
    push_default_route = true
  }
}
