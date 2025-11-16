# ============================================================================
# VPC et Private Networks avec IPAM automatique
# ============================================================================

# VPC principal pour le cluster Kubernetes
resource "scaleway_vpc" "kubernetes" {
  name   = "${var.cluster_name}-vpc"
  region = var.region
  tags   = local.common_tags
}

# Private Network pour le cluster avec IPAM automatique
resource "scaleway_vpc_private_network" "kubernetes" {
  name   = "${var.cluster_name}-pn"
  vpc_id = scaleway_vpc.kubernetes.id
  region = var.region
  tags   = local.common_tags

  # IPAM automatique - Scaleway alloue un /22 par défaut (1020 IPs)
  # Ou spécifier un CIDR personnalisé
  ipv4_subnet {
    subnet = var.kubernetes_cidr
  }
}

# VPC séparé pour le bastion (sécurité)
resource "scaleway_vpc" "bastion" {
  name   = "${var.cluster_name}-bastion-vpc"
  region = var.region
  tags   = local.common_tags
}

# Private Network pour le bastion
resource "scaleway_vpc_private_network" "bastion" {
  name   = "${var.cluster_name}-bastion-pn"
  vpc_id = scaleway_vpc.bastion.id
  region = var.region
  tags   = local.common_tags

  ipv4_subnet {
    subnet = var.bastion_cidr
  }
}

# ============================================================================
# Public Gateway pour NAT (accès internet sortant)
# ============================================================================

# IP publique flexible pour la Public Gateway
resource "scaleway_vpc_public_gateway_ip" "main" {
  zone = var.zone
  tags = local.common_tags
}

# Public Gateway en mode IPAM (obligatoire depuis nov 2025)
resource "scaleway_vpc_public_gateway" "main" {
  name            = "${var.cluster_name}-pgw"
  type            = var.public_gateway_type
  zone            = var.zone
  ip_id           = scaleway_vpc_public_gateway_ip.main.id
  
  # Bastion SSH intégré
  bastion_enabled = var.enable_bastion_on_gateway
  bastion_port    = var.bastion_ssh_port
  
  tags = local.common_tags
}

# Attachement de la Public Gateway au Private Network Kubernetes
resource "scaleway_vpc_gateway_network" "kubernetes" {
  gateway_id         = scaleway_vpc_public_gateway.main.id
  private_network_id = scaleway_vpc_private_network.kubernetes.id
  
  # NAT pour accès internet sortant
  enable_masquerade = true
  
  # Configuration IPAM
  ipam_config {
    push_default_route = true  # Route par défaut via la gateway
  }
  
  # DHCP géré automatiquement par le Private Network
  dhcp_id = null  # Mode IPAM uniquement
}

# ============================================================================
# Réservations IPAM pour IPs statiques (optionnel)
# ============================================================================

# Réserver des IPs pour les control planes (recommandé pour la stabilité)
resource "scaleway_ipam_ip" "control_plane" {
  count = var.control_plane_count

  address = cidrhost(var.kubernetes_cidr, 10 + count.index)
  source {
    private_network_id = scaleway_vpc_private_network.kubernetes.id
  }
  
  tags = concat(local.common_tags, [
    "role=control-plane",
    "node=cp-${count.index + 1}"
  ])
}

# Réserver des IPs pour les workers
resource "scaleway_ipam_ip" "workers" {
  count = var.worker_count

  address = cidrhost(var.kubernetes_cidr, 20 + count.index)
  source {
    private_network_id = scaleway_vpc_private_network.kubernetes.id
  }
  
  tags = concat(local.common_tags, [
    "role=worker",
    "node=worker-${count.index + 1}"
  ])
}

# ============================================================================
# VPC Peering (si bastion dans VPC séparé)
# ============================================================================

# Note: Le peering VPC n'est pas disponible en beta publique sur Scaleway
# Pour connecter bastion et cluster, utiliser:
# 1. Load Balancer public avec ACLs restrictives
# 2. Public Gateway avec bastion SSH
# 3. Bastion dans le même Private Network

# Pour ce projet, on utilisera la Public Gateway avec bastion SSH intégré
