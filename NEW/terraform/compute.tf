# ============================================================================
# Data sources pour les images
# ============================================================================

# Rechercher l'image Talos (alias stable créé dans chaque zone)
data "scaleway_instance_image" "talos" {
  name         = "talos-scaleway-${var.talos_version}" # ex: talos-scaleway-v1.11.5
  architecture = "x86_64"
  zone         = var.zone
  latest       = true
}

# Image Ubuntu pour le bastion (si besoin)
data "scaleway_instance_image" "ubuntu" {
  name         = "Ubuntu 22.04 Jammy Jellyfish"
  architecture = "x86_64"
  latest       = true
}

# ============================================================================
# Instances Control Plane (multi-AZ)
# ============================================================================

resource "scaleway_instance_server" "control_plane" {
  count = var.control_plane_count

  name              = "${var.cluster_name}-cp-${count.index + 1}"
  type              = var.control_plane_instance_type
  image             = data.scaleway_instance_image.talos.id

  # Distribution multi-AZ cyclique
  zone = local.availability_zones[count.index % length(local.availability_zones)]

  # Security group
  security_group_id = scaleway_instance_security_group.control_plane.id

  # Volume de boot - sbs_volume (Block Storage)
  root_volume {
    size_in_gb            = var.control_plane_disk_size
    volume_type           = "sbs_volume"  # Block Storage pour production
    delete_on_termination = true
  }

  # Attachement au Private Network
  # Note: IPAM allouera automatiquement une IP
  private_network {
    pn_id = scaleway_vpc_private_network.kubernetes.id
  }

  # Cloud-init sera géré par Talos lors de l'apply-config
  # Pas de user_data ici

  tags = concat(local.common_tags, [
    "role=control-plane",
    "az=${local.availability_zones[count.index % length(local.availability_zones)]}",
    "node-index=${count.index + 1}"
  ])

  # Attendre que le réseau soit prêt
  depends_on = [
    scaleway_vpc_gateway_network.kubernetes
  ]
}

# ============================================================================
# Instances Workers (multi-AZ)
# ============================================================================

resource "scaleway_instance_server" "workers" {
  count = var.worker_count

  name              = "${var.cluster_name}-worker-${count.index + 1}"
  type              = var.worker_instance_type
  image             = data.scaleway_instance_image.talos.id

  # Distribution multi-AZ cyclique
  zone = local.availability_zones[count.index % length(local.availability_zones)]

  # Security group
  security_group_id = scaleway_instance_security_group.workers.id

  # Volume de boot - sbs_volume (Block Storage)
  root_volume {
    size_in_gb            = var.worker_disk_size
    volume_type           = "sbs_volume"  # Block Storage pour production
    delete_on_termination = true
  }

  # Attachement au Private Network
  # IPAM allouera automatiquement une IP
  private_network {
    pn_id = scaleway_vpc_private_network.kubernetes.id
  }

  tags = concat(local.common_tags, [
    "role=worker",
    "az=${local.availability_zones[count.index % length(local.availability_zones)]}",
    "node-index=${count.index + 1}"
  ])

  depends_on = [
    scaleway_vpc_gateway_network.kubernetes
  ]
}

# ============================================================================
# Instance Bastion (optionnel)
# ============================================================================

resource "scaleway_instance_server" "bastion" {
  count = var.enable_bastion_instance ? 1 : 0

  name              = "${var.cluster_name}-bastion"
  type              = var.bastion_instance_type
  image             = data.scaleway_instance_image.ubuntu.id
  zone              = var.zone

  # Security group
  security_group_id = scaleway_instance_security_group.bastion[0].id

  # Volume de boot - sbs_volume suffit pour le bastion
  root_volume {
    size_in_gb            = 20
    volume_type           = "sbs_volume"  # Block Storage pour bastion
    delete_on_termination = true
  }

  # IP publique flexible
  ip_id = scaleway_instance_ip.bastion[0].id

  # Cloud-init pour installer les outils
  user_data = {
    cloud-init = templatefile("${path.module}/templates/bastion-cloud-init.yaml", {
      cluster_name  = var.cluster_name
      talos_version = var.talos_version
    })
  }

  tags = concat(local.common_tags, [
    "role=bastion"
  ])
}

# IP publique pour le bastion
resource "scaleway_instance_ip" "bastion" {
  count = var.enable_bastion_instance ? 1 : 0
  zone  = var.zone
  tags  = local.common_tags
}

# ============================================================================
# Outputs locaux
# ============================================================================

locals {
  # IP privées des control planes
  control_plane_ips = [
    for server in scaleway_instance_server.control_plane :
    [
      for pn in server.private_network :
      pn.ip
      if pn.pn_id == scaleway_vpc_private_network.kubernetes.id
    ][0]
  ]

  # IP privées des workers
  worker_ips = [
    for server in scaleway_instance_server.workers :
    [
      for pn in server.private_network :
      pn.ip
      if pn.pn_id == scaleway_vpc_private_network.kubernetes.id
    ][0]
  ]

  # Mapping AZ -> Instances pour visualisation (control planes)
  control_plane_by_az = {
    for idx, server in scaleway_instance_server.control_plane :
    local.availability_zones[idx % length(local.availability_zones)] => {
      name       = server.name
      private_ip = [
        for pn in server.private_network :
        pn.ip
        if pn.pn_id == scaleway_vpc_private_network.kubernetes.id
      ][0]
    }...
  }

  # Mapping AZ -> Instances pour visualisation (workers)
  workers_by_az = {
    for idx, server in scaleway_instance_server.workers :
    local.availability_zones[idx % length(local.availability_zones)] => {
      name       = server.name
      private_ip = [
        for pn in server.private_network :
        pn.ip
        if pn.pn_id == scaleway_vpc_private_network.kubernetes.id
      ][0]
    }...
  }
}
