# ============================================================================
# Configuration Terraform pour Talos Kubernetes sur Scaleway
# Copiez ce fichier en terraform.tfvars et ajustez les valeurs
# ============================================================================

# Credentials Scaleway (ou utiliser les variables d'environnement)
# scw_access_key = "SCWXXXXXXXXXXXXXXXXX"
# scw_secret_key = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
# scw_project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

# Région et zone
region = "fr-par"
zone   = "fr-par-1"

# Nom du cluster
cluster_name = "talos-prod"
environment  = "production"

# Version Talos (doit correspondre à l'image créée par Packer)
talos_version = "v1.11.5"

# ============================================================================
# Réseau
# ============================================================================

# CIDR du Private Network Kubernetes (IPAM automatique)
# Par défaut /22 = 1020 IPs utilisables
kubernetes_cidr = "10.0.0.0/22"

# CIDR du bastion (si instance dédiée)
bastion_cidr = "10.100.0.0/24"

# Multi-AZ (PAR-1, PAR-2, PAR-3)
enable_multi_az = true

# ============================================================================
# Control Planes
# ============================================================================

# Nombre de control planes (impair pour quorum etcd)
control_plane_count = 3

# Type d'instance
# DEV1-M: 3 vCPU, 4 GB RAM (dev/test)
# PRO2-S: 2 vCPU, 8 GB RAM (production, recommandé)
# PRO2-M: 4 vCPU, 16 GB RAM (production, charge élevée)
control_plane_instance_type = "PRO2-S"

# Taille du disque (GB)
control_plane_disk_size = 50

# ============================================================================
# Workers
# ============================================================================

# Nombre de workers
worker_count = 3

# Type d'instance
# PRO2-M: 4 vCPU, 16 GB RAM (recommandé)
# PRO2-L: 8 vCPU, 32 GB RAM (charge élevée)
# POP2-4C-16G: 4 vCPU dédiés, 16 GB RAM (haute performance)
worker_instance_type = "PRO2-M"

# Taille du disque (GB)
worker_disk_size = 100

# ============================================================================
# Load Balancer
# ============================================================================

# Type de Load Balancer
# LB-S: Petit (jusqu'à 500 conn/s) - recommandé pour API K8s
# LB-GP-M: Moyen (jusqu'à 5000 conn/s)
# LB-GP-L: Grand (jusqu'à 10000 conn/s)
load_balancer_type = "LB-S"

# Exposer l'API Kubernetes publiquement (déconseillé en production)
expose_k8s_api_publicly = false

# Exposer l'API Talos via le Load Balancer
expose_talos_api = true

# Activer les ACLs sur le Load Balancer (recommandé si public)
enable_lb_acls = true

# ============================================================================
# Public Gateway (NAT)
# ============================================================================

# Type de Public Gateway
# VPC-GW-S: 1 Gbps (suffisant pour la plupart des cas)
# VPC-GW-M: 2 Gbps
# VPC-GW-L: 3 Gbps
public_gateway_type = "VPC-GW-S"

# Activer le bastion SSH sur la Public Gateway (recommandé)
enable_bastion_on_gateway = true

# Port SSH du bastion
bastion_ssh_port = 61000

# ============================================================================
# Bastion (instance dédiée optionnelle)
# ============================================================================

# Créer une instance bastion dédiée en plus du bastion sur gateway
enable_bastion_instance = false

# Type d'instance pour le bastion
bastion_instance_type = "DEV1-S"

# CIDR autorisé pour SSH (vide = votre IP publique)
# Exemple: "203.0.113.0/24"
bastion_allowed_cidr = ""

# ============================================================================
# Tags additionnels
# ============================================================================

additional_tags = [
  "owner=platform-team",
  "cost-center=engineering",
  "backup=daily"
]

# ============================================================================
# Notes importantes
# ============================================================================

# 1. Multi-AZ:
#    - PAR-1, PAR-2, PAR-3 sont utilisés en rotation
#    - Les volumes Block Storage sont zonaux (non migrables)
#    - Les Private Networks sont régionaux (couvrent toutes les AZ)
#
# 2. IPAM:
#    - Les IPs privées sont allouées automatiquement par Scaleway
#    - Les IPs des control planes et workers sont réservées via IPAM
#    - DNS interne: <resource-name>.<private-network-name>.internal
#
# 3. Coûts estimés (fr-par, nov 2025):
#    - 3x PRO2-S (CP): ~0.165€/h
#    - 3x PRO2-M (workers): ~0.495€/h
#    - LB-S: ~0.02€/h
#    - VPC-GW-S: ~0.02€/h
#    - Block Storage SBS 5K: ~0.00012€/GB/h
#    Total: ~0.70€/h soit ~504€/mois
#
# 4. Limitations Scaleway:
#    - Security Groups filtrent uniquement le trafic PUBLIC
#    - Pour filtrer le trafic PRIVÉ, utiliser les NACLs (beta)
#    - SMTP ports bloqués par défaut (demande nécessaire)
