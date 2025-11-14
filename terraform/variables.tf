# ============================================================================
# Credentials et Projet Scaleway
# ============================================================================

variable "scw_access_key" {
  description = "Scaleway Access Key"
  type        = string
  sensitive   = true
}

variable "scw_secret_key" {
  description = "Scaleway Secret Key"
  type        = string
  sensitive   = true
}

variable "scw_project_id" {
  description = "Scaleway Project ID"
  type        = string
}

variable "region" {
  description = "Région Scaleway"
  type        = string
  default     = "fr-par"
  
  validation {
    condition     = contains(["fr-par", "nl-ams", "pl-waw"], var.region)
    error_message = "La région doit être fr-par, nl-ams ou pl-waw."
  }
}

variable "zone" {
  description = "Zone par défaut Scaleway"
  type        = string
  default     = "fr-par-1"
}

# ============================================================================
# Configuration du Cluster
# ============================================================================

variable "cluster_name" {
  description = "Nom du cluster Kubernetes"
  type        = string
  default     = "talos-prod"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.cluster_name))
    error_message = "Le nom du cluster doit contenir uniquement des minuscules, chiffres et tirets."
  }
}

variable "environment" {
  description = "Environnement (production, staging, dev)"
  type        = string
  default     = "production"
}

variable "talos_version" {
  description = "Version de Talos Linux"
  type        = string
  default     = "v1.11.5"
}

# ============================================================================
# Configuration Réseau
# ============================================================================

variable "kubernetes_cidr" {
  description = "CIDR pour le Private Network Kubernetes (IPAM automatique si non spécifié)"
  type        = string
  default     = "10.0.0.0/22"  # 1020 IPs utilisables
  
  validation {
    condition     = can(cidrhost(var.kubernetes_cidr, 0))
    error_message = "Le CIDR doit être valide (ex: 10.0.0.0/22)."
  }
}

variable "bastion_cidr" {
  description = "CIDR pour le Private Network du bastion"
  type        = string
  default     = "10.100.0.0/24"
}

variable "enable_multi_az" {
  description = "Activer le déploiement multi-AZ (PAR-1, PAR-2, PAR-3)"
  type        = bool
  default     = true
}

# ============================================================================
# Configuration Control Plane
# ============================================================================

variable "control_plane_count" {
  description = "Nombre de nœuds control plane (recommandé: 3 pour HA)"
  type        = number
  default     = 3
  
  validation {
    condition     = var.control_plane_count >= 1 && var.control_plane_count <= 5 && var.control_plane_count % 2 == 1
    error_message = "Le nombre de control planes doit être impair (1, 3 ou 5) pour le quorum etcd."
  }
}

variable "control_plane_instance_type" {
  description = "Type d'instance pour les control planes"
  type        = string
  default     = "PRO2-S"  # 2 vCPU, 8 GB RAM
  
  validation {
    condition     = can(regex("^(DEV1-[SML]|GP1-[XSLM]|PRO2-[XSLM]|POP2-.*C-.*)$", var.control_plane_instance_type))
    error_message = "Type d'instance invalide. Exemples: DEV1-M, PRO2-S, POP2-4C-16G."
  }
}

variable "control_plane_disk_size" {
  description = "Taille du disque pour les control planes (GB)"
  type        = number
  default     = 50
  
  validation {
    condition     = var.control_plane_disk_size >= 20 && var.control_plane_disk_size <= 10000
    error_message = "La taille du disque doit être entre 20 GB et 10 TB."
  }
}

# ============================================================================
# Configuration Workers
# ============================================================================

variable "worker_count" {
  description = "Nombre de nœuds workers"
  type        = number
  default     = 3
  
  validation {
    condition     = var.worker_count >= 0 && var.worker_count <= 50
    error_message = "Le nombre de workers doit être entre 0 et 50."
  }
}

variable "worker_instance_type" {
  description = "Type d'instance pour les workers"
  type        = string
  default     = "PRO2-M"  # 4 vCPU, 16 GB RAM
}

variable "worker_disk_size" {
  description = "Taille du disque pour les workers (GB)"
  type        = number
  default     = 100
}

# ============================================================================
# Configuration Load Balancer
# ============================================================================

variable "load_balancer_type" {
  description = "Type de Load Balancer Scaleway"
  type        = string
  default     = "LB-S"  # Petit LB, suffisant pour API K8s
  
  validation {
    condition     = contains(["LB-S", "LB-GP-M", "LB-GP-L"], var.load_balancer_type)
    error_message = "Le type de LB doit être LB-S, LB-GP-M ou LB-GP-L."
  }
}

variable "expose_k8s_api_publicly" {
  description = "Exposer l'API Kubernetes publiquement (déconseillé en production)"
  type        = bool
  default     = false
}

variable "expose_talos_api" {
  description = "Exposer l'API Talos via le Load Balancer"
  type        = bool
  default     = true
}

variable "enable_lb_acls" {
  description = "Activer les ACLs sur le Load Balancer"
  type        = bool
  default     = true
}

# ============================================================================
# Configuration Public Gateway
# ============================================================================

variable "public_gateway_type" {
  description = "Type de Public Gateway"
  type        = string
  default     = "VPC-GW-S"  # 1 Gbps
  
  validation {
    condition     = contains(["VPC-GW-S", "VPC-GW-M", "VPC-GW-L"], var.public_gateway_type)
    error_message = "Le type de gateway doit être VPC-GW-S, VPC-GW-M ou VPC-GW-L."
  }
}

variable "enable_bastion_on_gateway" {
  description = "Activer le bastion SSH sur la Public Gateway"
  type        = bool
  default     = true
}

variable "bastion_ssh_port" {
  description = "Port SSH pour le bastion sur la Public Gateway"
  type        = number
  default     = 61000
}

# ============================================================================
# Configuration Bastion (instance dédiée optionnelle)
# ============================================================================

variable "enable_bastion_instance" {
  description = "Créer une instance bastion dédiée (en plus du bastion sur gateway)"
  type        = bool
  default     = false
}

variable "bastion_instance_type" {
  description = "Type d'instance pour le bastion"
  type        = string
  default     = "DEV1-S"
}

variable "bastion_allowed_cidr" {
  description = "CIDR autorisé pour SSH (laisser vide pour auto-détecter votre IP)"
  type        = string
  default     = ""
}

# ============================================================================
# Tags additionnels
# ============================================================================

variable "additional_tags" {
  description = "Tags additionnels à appliquer à toutes les ressources"
  type        = list(string)
  default     = []
}

# ============================================================================
# Notes sur les types d'instances Scaleway
# ============================================================================

# Types d'instances recommandés pour Kubernetes:
#
# Development/Test:
#   - DEV1-S: 2 vCPU, 2 GB RAM (€0.01/h)
#   - DEV1-M: 3 vCPU, 4 GB RAM (€0.02/h)
#   - DEV1-L: 4 vCPU, 8 GB RAM (€0.04/h)
#
# Production (vCPU partagé):
#   - PRO2-XXS: 0.5 vCPU, 2 GB RAM
#   - PRO2-XS:  1 vCPU, 4 GB RAM
#   - PRO2-S:   2 vCPU, 8 GB RAM (recommandé control plane)
#   - PRO2-M:   4 vCPU, 16 GB RAM (recommandé workers)
#   - PRO2-L:   8 vCPU, 32 GB RAM
#
# Production (vCPU dédié):
#   - POP2-2C-8G:   2 vCPU, 8 GB RAM
#   - POP2-4C-16G:  4 vCPU, 16 GB RAM
#   - POP2-8C-32G:  8 vCPU, 32 GB RAM
#   - POP2-16C-64G: 16 vCPU, 64 GB RAM
#
# ARM (PAR-2 uniquement):
#   - COPARM1-2C-8G: 2 vCPU ARM, 8 GB RAM
#   - COPARM1-4C-16G: 4 vCPU ARM, 16 GB RAM
