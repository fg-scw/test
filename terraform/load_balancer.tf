# ============================================================================
# Load Balancer pour l'API Kubernetes (Haute Disponibilité)
# ============================================================================

# Load Balancer privé (internal) attaché au Private Network
resource "scaleway_lb" "kubernetes" {
  name        = "${var.cluster_name}-k8s-lb"
  description = "Load Balancer for Kubernetes API (control plane)"
  type        = var.load_balancer_type
  zone        = var.zone
  
  # LB privé (sans IP publique)
  assign_flexible_ip = var.expose_k8s_api_publicly
  
  # Attachement au Private Network pour atteindre les backends
  private_network {
    private_network_id = scaleway_vpc_private_network.kubernetes.id
    dhcp_config        = true  # Obtenir une IP via IPAM
  }
  
  tags = concat(local.common_tags, [
    "role=kubernetes-api-lb"
  ])
}

# IP publique flexible (si exposition publique demandée)
resource "scaleway_lb_ip" "kubernetes" {
  count = var.expose_k8s_api_publicly ? 1 : 0
  zone  = var.zone
  tags  = local.common_tags
}

# Backend pour l'API Kubernetes (port 6443)
resource "scaleway_lb_backend" "k8s_api" {
  lb_id            = scaleway_lb.kubernetes.id
  name             = "k8s-api-backend"
  forward_protocol = "tcp"
  forward_port     = 6443
  
  # Health check TCP sur l'API Kubernetes
  health_check_tcp {
    port = 6443
  }
  
  # Timeout
  timeout_server  = "10s"
  timeout_connect = "5s"
  timeout_tunnel  = "0s"  # Pas de timeout pour les connexions persistantes
  
  # Algorithme de distribution
  # - roundrobin: distribution équitable
  # - leastconn: vers le serveur avec le moins de connexions
  on_marked_down_action = "on_marked_down_action_none"
}

# Frontend pour l'API Kubernetes
resource "scaleway_lb_frontend" "k8s_api" {
  lb_id        = scaleway_lb.kubernetes.id
  backend_id   = scaleway_lb_backend.k8s_api.id
  name         = "k8s-api-frontend"
  inbound_port = 6443
  
  # Timeout
  timeout_client = "10s"
}

# Attachement des control planes au backend
resource "scaleway_lb_backend_server" "control_planes_k8s_api" {
  count = var.control_plane_count

  backend_id = scaleway_lb_backend.k8s_api.id
  
  # IP privée du control plane
  server_ip = scaleway_instance_server.control_plane[count.index].private_network[0].ip
  name      = "${var.cluster_name}-cp-${count.index + 1}"
}

# ============================================================================
# Backend pour l'API Talos (port 50000) - Optionnel
# ============================================================================

resource "scaleway_lb_backend" "talos_api" {
  count = var.expose_talos_api ? 1 : 0

  lb_id            = scaleway_lb.kubernetes.id
  name             = "talos-api-backend"
  forward_protocol = "tcp"
  forward_port     = 50000
  
  # Health check TCP sur l'API Talos
  health_check_tcp {
    port = 50000
  }
  
  timeout_server  = "10s"
  timeout_connect = "5s"
  timeout_tunnel  = "0s"
}

# Frontend pour l'API Talos
resource "scaleway_lb_frontend" "talos_api" {
  count = var.expose_talos_api ? 1 : 0

  lb_id        = scaleway_lb.kubernetes.id
  backend_id   = scaleway_lb_backend.talos_api[0].id
  name         = "talos-api-frontend"
  inbound_port = 50000
  
  timeout_client = "10s"
}

# Attachement des control planes au backend Talos
resource "scaleway_lb_backend_server" "control_planes_talos_api" {
  count = var.expose_talos_api ? var.control_plane_count : 0

  backend_id = scaleway_lb_backend.talos_api[0].id
  server_ip  = scaleway_instance_server.control_plane[count.index].private_network[0].ip
  name       = "${var.cluster_name}-cp-${count.index + 1}-talos"
}

# ============================================================================
# ACLs pour restreindre l'accès (si LB public)
# ============================================================================

resource "scaleway_lb_acl" "k8s_api_allow_bastion" {
  count = var.expose_k8s_api_publicly && var.enable_lb_acls ? 1 : 0

  frontend_id = scaleway_lb_frontend.k8s_api.id
  name        = "allow-bastion-only"
  
  # Autoriser uniquement depuis le bastion ou IP spécifiée
  action {
    type = "allow"
  }
  
  match {
    ip_subnet = [
      coalesce(var.bastion_allowed_cidr, local.my_ip_cidr)
    ]
  }
  
  index = 0
}

# Deny all par défaut (si ACLs activées)
resource "scaleway_lb_acl" "k8s_api_deny_all" {
  count = var.expose_k8s_api_publicly && var.enable_lb_acls ? 1 : 0

  frontend_id = scaleway_lb_frontend.k8s_api.id
  name        = "deny-all"
  
  action {
    type = "deny"
  }
  
  match {
    ip_subnet = ["0.0.0.0/0"]
  }
  
  index = 100  # Évalué en dernier
}

# ============================================================================
# Outputs locaux
# ============================================================================

locals {
  k8s_api_endpoint = var.expose_k8s_api_publicly ? (
    scaleway_lb.kubernetes.ip_address  # IP publique du LB
  ) : (
    scaleway_lb.kubernetes.private_network[0].static_ip_address[0]  # IP privée
  )
}

# ============================================================================
# Notes sur le Load Balancer Scaleway
# ============================================================================

# 1. Types de Load Balancers disponibles:
#    - LB-S: Petit (jusqu'à 500 connexions/s)
#    - LB-GP-M: Général (jusqu'à 5000 connexions/s)
#    - LB-GP-L: Grand (jusqu'à 10000 connexions/s)
#
# 2. Load Balancer privé vs public:
#    - assign_flexible_ip = false : LB privé, uniquement dans le Private Network
#    - assign_flexible_ip = true  : LB public avec IP flexible
#
# 3. Le LB peut atteindre les backends dans le Private Network via VPC routing
#
# 4. Health checks:
#    - tcp: simple vérification de port
#    - http/https: vérification avec path et codes HTTP
#
# 5. Multi-AZ:
#    - Le LB est zonal mais peut distribuer vers backends multi-AZ
#    - Les backends dans d'autres zones sont accessibles via le Private Network régional
