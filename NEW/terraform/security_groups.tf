# ============================================================================
# Security Groups pour le cluster Kubernetes
# ============================================================================

# Security Group pour les Control Planes
resource "scaleway_instance_security_group" "control_plane" {
  name        = "${var.cluster_name}-cp-sg"
  description = "Security group for Talos control plane nodes"
  
  # Politique par défaut: deny all inbound, allow all outbound
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  stateful                = true
  
  tags = local.common_tags
}

# Règles inbound pour Control Plane
# API Kubernetes depuis Load Balancer
resource "scaleway_instance_security_group_rules" "control_plane_k8s_api" {
  security_group_id = scaleway_instance_security_group.control_plane.id
  
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 6443
    # Le LB est dans le même Private Network
    ip_range = var.kubernetes_cidr
  }
}

# API Talos depuis Load Balancer et bastion
resource "scaleway_instance_security_group_rules" "control_plane_talos_api" {
  security_group_id = scaleway_instance_security_group.control_plane.id
  
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 50000
    ip_range = var.kubernetes_cidr
  }
}

# etcd inter-control-planes
resource "scaleway_instance_security_group_rules" "control_plane_etcd" {
  security_group_id = scaleway_instance_security_group.control_plane.id
  
  # etcd client
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 2379
    ip_range = var.kubernetes_cidr
  }
  
  # etcd peer
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 2380
    ip_range = var.kubernetes_cidr
  }
}

# Kubelet API
resource "scaleway_instance_security_group_rules" "control_plane_kubelet" {
  security_group_id = scaleway_instance_security_group.control_plane.id
  
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 10250
    ip_range = var.kubernetes_cidr
  }
}

# Cilium health checks et Hubble
resource "scaleway_instance_security_group_rules" "control_plane_cilium" {
  security_group_id = scaleway_instance_security_group.control_plane.id
  
  # Cilium health
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 4240
    ip_range = var.kubernetes_cidr
  }
  
  # Hubble server
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port_range = "4244-4245"
    ip_range = var.kubernetes_cidr
  }
  
  # VXLAN (si overlay network)
  inbound_rule {
    action   = "accept"
    protocol = "UDP"
    port     = 8472
    ip_range = var.kubernetes_cidr
  }
}

# ICMP pour ping
resource "scaleway_instance_security_group_rules" "control_plane_icmp" {
  security_group_id = scaleway_instance_security_group.control_plane.id
  
  inbound_rule {
    action   = "accept"
    protocol = "ICMP"
    ip_range = var.kubernetes_cidr
  }
}

# ============================================================================
# Security Group pour les Workers
# ============================================================================

resource "scaleway_instance_security_group" "workers" {
  name        = "${var.cluster_name}-workers-sg"
  description = "Security group for Talos worker nodes"
  
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  stateful                = true
  
  tags = local.common_tags
}

# Règles inbound pour Workers
# Kubelet API
resource "scaleway_instance_security_group_rules" "workers_kubelet" {
  security_group_id = scaleway_instance_security_group.workers.id
  
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 10250
    ip_range = var.kubernetes_cidr
  }
}

# NodePort Services (30000-32767)
resource "scaleway_instance_security_group_rules" "workers_nodeport" {
  security_group_id = scaleway_instance_security_group.workers.id
  
  inbound_rule {
    action     = "accept"
    protocol   = "TCP"
    port_range = "30000-32767"
    ip_range   = var.kubernetes_cidr
  }
}

# Cilium health et Hubble
resource "scaleway_instance_security_group_rules" "workers_cilium" {
  security_group_id = scaleway_instance_security_group.workers.id
  
  # Cilium health
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 4240
    ip_range = var.kubernetes_cidr
  }
  
  # Hubble
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port_range = "4244-4245"
    ip_range = var.kubernetes_cidr
  }
  
  # VXLAN
  inbound_rule {
    action   = "accept"
    protocol = "UDP"
    port     = 8472
    ip_range = var.kubernetes_cidr
  }
}

# API Talos pour gestion
resource "scaleway_instance_security_group_rules" "workers_talos_api" {
  security_group_id = scaleway_instance_security_group.workers.id
  
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 50000
    ip_range = var.kubernetes_cidr
  }
}

# ICMP
resource "scaleway_instance_security_group_rules" "workers_icmp" {
  security_group_id = scaleway_instance_security_group.workers.id
  
  inbound_rule {
    action   = "accept"
    protocol = "ICMP"
    ip_range = var.kubernetes_cidr
  }
}

# ============================================================================
# Security Group pour le Bastion (optionnel)
# ============================================================================

resource "scaleway_instance_security_group" "bastion" {
  count = var.enable_bastion_instance ? 1 : 0

  name        = "${var.cluster_name}-bastion-sg"
  description = "Security group for bastion host"
  
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  stateful                = true
  
  tags = local.common_tags
}

# SSH depuis IP autorisée
resource "scaleway_instance_security_group_rules" "bastion_ssh" {
  count = var.enable_bastion_instance ? 1 : 0

  security_group_id = scaleway_instance_security_group.bastion[0].id
  
  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 22
    ip_range = coalesce(var.bastion_allowed_cidr, local.my_ip_cidr)
  }
}

# ============================================================================
# Notes sur les Security Groups Scaleway
# ============================================================================

# 1. Les Security Groups sont STATEFUL par défaut
#    - Le retour du trafic est automatiquement autorisé
#
# 2. Politique par défaut recommandée:
#    - inbound: drop (deny all)
#    - outbound: accept (allow all)
#
# 3. Les règles s'appliquent uniquement au trafic PUBLIC
#    - Le trafic entre instances du même Private Network passe
#      par le réseau privé et n'est pas filtré par les SG
#
# 4. Pour filtrer le trafic PRIVÉ entre Private Networks:
#    - Utiliser les NACLs (Network ACLs) du VPC
#    - Actuellement en beta publique
#
# 5. SMTP ports (25, 465, 587) sont bloqués par défaut par Scaleway
#    - Nécessite une demande explicite pour les débloquer
