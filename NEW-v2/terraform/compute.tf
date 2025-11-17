# ============================================================================
# Data Sources
# ============================================================================

data "scaleway_instance_image" "talos" {
  name         = var.talos_image_name != "" ? var.talos_image_name : "talos-scaleway-${var.talos_version}"
  architecture = "x86_64"
  zone         = var.zone
  latest       = true
}

# ============================================================================
# Control Plane Instances
# ============================================================================

resource "scaleway_instance_server" "control_plane" {
  count = var.control_plane_count

  name              = "${var.cluster_name}-cp-${count.index + 1}"
  type              = var.control_plane_instance_type
  image             = data.scaleway_instance_image.talos.id
  zone              = var.zone
  security_group_id = scaleway_instance_security_group.control_plane.id

  root_volume {
    size_in_gb            = var.control_plane_disk_size
    volume_type           = "sbs_volume"
    delete_on_termination = true
  }

  private_network {
    pn_id = scaleway_vpc_private_network.kubernetes.id
  }

  tags = concat(local.common_tags, [
    "role=control-plane",
    "node-index=${count.index + 1}"
  ])

  depends_on = [
    scaleway_vpc_gateway_network.kubernetes
  ]
}

# ============================================================================
# Worker Instances
# ============================================================================

resource "scaleway_instance_server" "workers" {
  count = var.worker_count

  name              = "${var.cluster_name}-worker-${count.index + 1}"
  type              = var.worker_instance_type
  image             = data.scaleway_instance_image.talos.id
  zone              = var.zone
  security_group_id = scaleway_instance_security_group.workers.id

  root_volume {
    size_in_gb            = var.worker_disk_size
    volume_type           = "sbs_volume"
    delete_on_termination = true
  }

  private_network {
    pn_id = scaleway_vpc_private_network.kubernetes.id
  }

  tags = concat(local.common_tags, [
    "role=worker",
    "node-index=${count.index + 1}"
  ])

  depends_on = [
    scaleway_vpc_gateway_network.kubernetes
  ]
}

# ============================================================================
# Data sources pour récupérer les IPs privées via IPAM
# ============================================================================

data "scaleway_ipam_ip" "control_plane" {
  count = var.control_plane_count

  mac_address = scaleway_instance_server.control_plane[count.index].private_network[0].mac_address
  type        = "ipv4"
}

data "scaleway_ipam_ip" "workers" {
  count = var.worker_count

  mac_address = scaleway_instance_server.workers[count.index].private_network[0].mac_address
  type        = "ipv4"
}

# ============================================================================
# Locals pour les IPs privées
# ============================================================================

locals {
  control_plane_ips = [
    for ipam in data.scaleway_ipam_ip.control_plane :
    split("/", ipam.address)[0]
  ]

  worker_ips = [
    for ipam in data.scaleway_ipam_ip.workers :
    split("/", ipam.address)[0]
  ]
}