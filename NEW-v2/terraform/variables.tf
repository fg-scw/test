# ============================================================================
# Scaleway Configuration
# ============================================================================

variable "region" {
  description = "Scaleway region"
  type        = string
  default     = "fr-par"
  
  validation {
    condition     = contains(["fr-par", "nl-ams", "pl-waw"], var.region)
    error_message = "Region must be fr-par, nl-ams, or pl-waw."
  }
}

variable "zone" {
  description = "Scaleway availability zone"
  type        = string
  default     = "fr-par-1"
  
  validation {
    condition     = can(regex("^(fr-par|nl-ams|pl-waw)-[1-3]$", var.zone))
    error_message = "Zone must be in format <region>-<number> (e.g., fr-par-1)."
  }
}

variable "scw_project_id" {
  description = "Scaleway Project ID (can also be set via SCW_DEFAULT_PROJECT_ID env var)"
  type        = string
  default     = ""
}

# ============================================================================
# Cluster Configuration
# ============================================================================

variable "cluster_name" {
  description = "Kubernetes cluster name"
  type        = string
  default     = "talos-k8s"
  
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.cluster_name))
    error_message = "Cluster name must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "environment" {
  description = "Environment (production, staging, dev)"
  type        = string
  default     = "production"
}

variable "talos_version" {
  description = "Talos Linux version"
  type        = string
  default     = "v1.11.5"
}

# ============================================================================
# Network Configuration
# ============================================================================

variable "private_network_cidr" {
  description = "CIDR for the Private Network (IPAM will auto-allocate if not specified)"
  type        = string
  default     = "10.0.0.0/22"
  
  validation {
    condition     = can(cidrhost(var.private_network_cidr, 0))
    error_message = "CIDR must be valid (e.g., 10.0.0.0/22)."
  }
}

# ============================================================================
# Control Plane Configuration
# ============================================================================

variable "control_plane_count" {
  description = "Number of control plane nodes (must be odd for etcd quorum)"
  type        = number
  default     = 3
  
  validation {
    condition     = var.control_plane_count >= 1 && var.control_plane_count <= 5 && var.control_plane_count % 2 == 1
    error_message = "Control plane count must be odd (1, 3, or 5) for etcd quorum."
  }
}

variable "control_plane_instance_type" {
  description = "Instance type for control plane nodes"
  type        = string
  default     = "PRO2-S"
  
  validation {
    condition     = can(regex("^(DEV1-[SML]|GP1-[XSLM]|PRO2-[XSLM]|POP2-.*C-.*)$", var.control_plane_instance_type))
    error_message = "Invalid instance type. Examples: DEV1-M, PRO2-S, POP2-4C-16G."
  }
}

variable "control_plane_disk_size" {
  description = "Root disk size for control plane nodes (GB)"
  type        = number
  default     = 50
  
  validation {
    condition     = var.control_plane_disk_size >= 20 && var.control_plane_disk_size <= 10000
    error_message = "Disk size must be between 20 GB and 10 TB."
  }
}

# ============================================================================
# Worker Configuration
# ============================================================================

variable "worker_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 3
  
  validation {
    condition     = var.worker_count >= 0 && var.worker_count <= 50
    error_message = "Worker count must be between 0 and 50."
  }
}

variable "worker_instance_type" {
  description = "Instance type for worker nodes"
  type        = string
  default     = "PRO2-M"
}

variable "worker_disk_size" {
  description = "Root disk size for worker nodes (GB)"
  type        = number
  default     = 100
}

# ============================================================================
# Load Balancer Configuration
# ============================================================================

variable "load_balancer_type" {
  description = "Load Balancer type for Kubernetes API"
  type        = string
  default     = "LB-S"
  
  validation {
    condition     = contains(["LB-S", "LB-GP-M", "LB-GP-L"], var.load_balancer_type)
    error_message = "LB type must be LB-S, LB-GP-M, or LB-GP-L."
  }
}

variable "expose_k8s_api_publicly" {
  description = "Expose Kubernetes API publicly (not recommended for production)"
  type        = bool
  default     = false
}

variable "expose_talos_api" {
  description = "Expose Talos API via Load Balancer (port 50000)"
  type        = bool
  default     = true
}

# ============================================================================
# Public Gateway Configuration
# ============================================================================

variable "public_gateway_type" {
  description = "Public Gateway type for NAT"
  type        = string
  default     = "VPC-GW-S"
  
  validation {
    condition     = contains(["VPC-GW-S", "VPC-GW-M", "VPC-GW-L"], var.public_gateway_type)
    error_message = "Gateway type must be VPC-GW-S, VPC-GW-M, or VPC-GW-L."
  }
}

variable "enable_bastion_on_gateway" {
  description = "Enable SSH bastion on Public Gateway"
  type        = bool
  default     = true
}

variable "bastion_ssh_port" {
  description = "SSH port for bastion on Public Gateway"
  type        = number
  default     = 61000
}

variable "bastion_allowed_cidr" {
  description = "CIDR allowed to access bastion (leave empty for 0.0.0.0/0)"
  type        = string
  default     = ""
}

# ============================================================================
# Additional Configuration
# ============================================================================

variable "additional_tags" {
  description = "Additional tags to apply to all resources"
  type        = list(string)
  default     = []
}

variable "talos_image_name" {
  description = "Name of the Talos image created by Packer (will be prefixed with talos-scaleway-)"
  type        = string
  default     = ""
}

# ============================================================================
# Bootstrap Bastion Configuration
# ============================================================================

variable "enable_bootstrap_bastion" {
  description = "Créer une instance bastion temporaire pour bootstrapper le cluster"
  type        = bool
  default     = true
}

variable "bastion_instance_type" {
  description = "Type d'instance pour le bastion de bootstrap"
  type        = string
  default     = "DEV1-S"
}
