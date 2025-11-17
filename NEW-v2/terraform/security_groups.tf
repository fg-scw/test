# ============================================================================
# Security Groups
# ============================================================================

# Security Group pour Control Plane
resource "scaleway_instance_security_group" "control_plane" {
  name                    = "${var.cluster_name}-cp-sg"
  description             = "Security group for Talos control plane nodes"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  stateful                = true
  zone                    = var.zone
  tags                    = local.common_tags

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 6443
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 50000
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 2379
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 2380
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 10250
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 4240
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action     = "accept"
    protocol   = "TCP"
    port_range = "4244-4245"
    ip_range   = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "UDP"
    port     = 8472
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "ICMP"
    ip_range = var.private_network_cidr
  }
}

# Security Group pour Workers
resource "scaleway_instance_security_group" "workers" {
  name                    = "${var.cluster_name}-workers-sg"
  description             = "Security group for Talos worker nodes"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  stateful                = true
  zone                    = var.zone
  tags                    = local.common_tags

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 10250
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action     = "accept"
    protocol   = "TCP"
    port_range = "30000-32767"
    ip_range   = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 4240
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action     = "accept"
    protocol   = "TCP"
    port_range = "4244-4245"
    ip_range   = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "UDP"
    port     = 8472
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 50000
    ip_range = var.private_network_cidr
  }

  inbound_rule {
    action   = "accept"
    protocol = "ICMP"
    ip_range = var.private_network_cidr
  }
}
