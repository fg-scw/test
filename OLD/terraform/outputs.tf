# ============================================================================
# Outputs essentiels pour le déploiement Talos
# ============================================================================

# API Kubernetes endpoint
output "kubernetes_api_endpoint" {
  description = "Endpoint de l'API Kubernetes (Load Balancer)"
  value       = "https://${local.k8s_api_endpoint}:6443"
}

output "kubernetes_api_ip" {
  description = "IP du Load Balancer pour l'API Kubernetes"
  value       = local.k8s_api_endpoint
}

# Talos API endpoint
output "talos_api_endpoint" {
  description = "Endpoint de l'API Talos (Load Balancer)"
  value       = var.expose_talos_api ? "https://${local.k8s_api_endpoint}:50000" : "N/A - Talos API not exposed"
}

# ============================================================================
# IPs des nœuds
# ============================================================================

output "control_plane_ips" {
  description = "IPs privées des control planes"
  value       = local.control_plane_ips
}

output "worker_ips" {
  description = "IPs privées des workers"
  value       = local.worker_ips
}

# Distribution par AZ
output "control_plane_distribution" {
  description = "Distribution des control planes par AZ"
  value       = local.control_plane_by_az
}

output "workers_distribution" {
  description = "Distribution des workers par AZ"
  value       = local.workers_by_az
}

# ============================================================================
# Réseau
# ============================================================================

output "vpc_id" {
  description = "ID du VPC Kubernetes"
  value       = scaleway_vpc.kubernetes.id
}

output "private_network_id" {
  description = "ID du Private Network Kubernetes"
  value       = scaleway_vpc_private_network.kubernetes.id
}

output "private_network_cidr" {
  description = "CIDR du Private Network"
  value       = scaleway_vpc_private_network.kubernetes.ipv4_subnet[0].subnet
}

output "public_gateway_ip" {
  description = "IP publique de la Public Gateway (NAT)"
  value       = scaleway_vpc_public_gateway_ip.main.address
}

# ============================================================================
# Bastion
# ============================================================================

output "bastion_ssh_command" {
  description = "Commande SSH pour se connecter au bastion (via Public Gateway)"
  value = var.enable_bastion_on_gateway ? (
    "ssh -J bastion@${scaleway_vpc_public_gateway_ip.main.address}:${var.bastion_ssh_port} root@<CONTROL_PLANE_IP>"
  ) : "Bastion not enabled on gateway"
}

output "bastion_instance_ip" {
  description = "IP publique de l'instance bastion (si créée)"
  value       = var.enable_bastion_instance ? scaleway_instance_ip.bastion[0].address : "N/A"
}

# ============================================================================
# Instructions de déploiement
# ============================================================================

output "next_steps" {
  description = "Prochaines étapes pour déployer Talos"
  value = <<-EOT
  
  ╔══════════════════════════════════════════════════════════════╗
  ║           Infrastructure Scaleway déployée avec succès !     ║
  ╚══════════════════════════════════════════════════════════════╝
  
  📋 Prochaines étapes:
  
  1️⃣  Générer les configurations Talos:
     
     talosctl gen config ${var.cluster_name} https://${local.k8s_api_endpoint}:6443 \
       --output-dir _out \
       --with-docs=false \
       --with-examples=false
  
  2️⃣  Appliquer le patch Cilium (désactiver kube-proxy):
     
     talosctl --talosconfig _out/talosconfig machineconfig patch \
       _out/controlplane.yaml \
       --patch @../cilium-patch.yaml \
       -o _out/controlplane-patched.yaml
  
  3️⃣  Appliquer les configurations aux control planes:
     
     %{ for ip in local.control_plane_ips ~}
     talosctl apply-config --insecure --nodes ${ip} --file _out/controlplane-patched.yaml
     %{ endfor ~}
  
  4️⃣  Appliquer les configurations aux workers:
     
     %{ for ip in local.worker_ips ~}
     talosctl apply-config --insecure --nodes ${ip} --file _out/worker.yaml
     %{ endfor ~}
  
  5️⃣  Bootstrap etcd (premier control plane uniquement):
     
     talosctl --talosconfig _out/talosconfig bootstrap --nodes ${local.control_plane_ips[0]}
  
  6️⃣  Récupérer le kubeconfig:
     
     talosctl --talosconfig _out/talosconfig kubeconfig _out/kubeconfig \
       --nodes ${local.k8s_api_endpoint}
     
     export KUBECONFIG=$(pwd)/_out/kubeconfig
     kubectl get nodes
  
  7️⃣  Installer Cilium:
     
     helm repo add cilium https://helm.cilium.io/
     helm repo update
     
     helm install cilium cilium/cilium \
       --namespace kube-system \
       --set ipam.mode=kubernetes \
       --set kubeProxyReplacement=true \
       --set k8sServiceHost=${local.k8s_api_endpoint} \
       --set k8sServicePort=6443
  
  8️⃣  Vérifier le cluster:
     
     kubectl get nodes
     kubectl -n kube-system get pods
     cilium status --wait
  
  📚 Documentation complète: https://www.talos.dev/
  
  EOT
}

# ============================================================================
# Informations de debugging
# ============================================================================

output "debug_info" {
  description = "Informations de debugging"
  value = {
    region                = var.region
    zones_used            = local.availability_zones
    multi_az_enabled      = var.enable_multi_az
    control_plane_count   = var.control_plane_count
    worker_count          = var.worker_count
    lb_type               = var.load_balancer_type
    lb_is_public          = var.expose_k8s_api_publicly
    gateway_type          = var.public_gateway_type
    bastion_on_gateway    = var.enable_bastion_on_gateway
    bastion_instance      = var.enable_bastion_instance
  }
}

# ============================================================================
# Export pour scripts
# ============================================================================

output "control_plane_ips_json" {
  description = "IPs des control planes au format JSON"
  value       = jsonencode(local.control_plane_ips)
}

output "worker_ips_json" {
  description = "IPs des workers au format JSON"
  value       = jsonencode(local.worker_ips)
}
