# ============================================================================
# Load Balancer pour l'API Kubernetes
# ============================================================================

resource "scaleway_lb_ip" "kubernetes" {
  count = var.expose_k8s_api_publicly ? 1 : 0
  zone  = var.zone
  tags  = local.common_tags
}

resource "scaleway_lb" "kubernetes" {
  name               = "${var.cluster_name}-k8s-lb"
  description        = "Load Balancer for Kubernetes API"
  type               = var.load_balancer_type
  zone               = var.zone
  assign_flexible_ip = var.expose_k8s_api_publicly
  ip_id              = var.expose_k8s_api_publicly ? scaleway_lb_ip.kubernetes[0].id : null
  tags               = local.common_tags

  private_network {
    private_network_id = scaleway_vpc_private_network.kubernetes.id
  }
}

# Backend pour l'API Kubernetes (port 6443)
resource "scaleway_lb_backend" "k8s_api" {
  lb_id            = scaleway_lb.kubernetes.id
  name             = "k8s-api-backend"
  forward_protocol = "tcp"
  forward_port     = 6443
  server_ips       = local.control_plane_ips

  health_check_tcp {}

  timeout_server  = "10s"
  timeout_connect = "5s"
  timeout_tunnel  = "0s"

  on_marked_down_action = "none"
}

# Frontend pour l'API Kubernetes
resource "scaleway_lb_frontend" "k8s_api" {
  lb_id        = scaleway_lb.kubernetes.id
  backend_id   = scaleway_lb_backend.k8s_api.id
  name         = "k8s-api-frontend"
  inbound_port = 6443

  timeout_client = "10s"
}

# Backend pour l'API Talos (port 50000) - optionnel
resource "scaleway_lb_backend" "talos_api" {
  count = var.expose_talos_api ? 1 : 0

  lb_id            = scaleway_lb.kubernetes.id
  name             = "talos-api-backend"
  forward_protocol = "tcp"
  forward_port     = 50000
  server_ips       = local.control_plane_ips

  health_check_tcp {}

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

# ============================================================================
# Locals pour l'endpoint Kubernetes API
# ============================================================================

locals {
  # Si le LB est public, on utilise l'IP publique
  # Si le LB est privé, on utilise l'IP du premier control plane comme endpoint
  # L'IP privée du LB sera visible dans la console Scaleway après déploiement
  k8s_api_endpoint = var.expose_k8s_api_publicly ? (
    scaleway_lb.kubernetes.ip_address
  ) : (
    # En mode privé, pointer vers le premier control plane
    # L'utilisateur devra récupérer l'IP du LB depuis la console pour talosctl
    length(local.control_plane_ips) > 0 ? local.control_plane_ips[0] : ""
  )
}