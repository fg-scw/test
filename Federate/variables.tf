variable "ssh_public_key" {
  description = "SSH public key used for root access on all nodes."
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path to the SSH private key matching ssh_public_key."
  type        = string
  default     = "~/.ssh/id_rsa"
}

variable "controller_instance_type" {
  description = "Scaleway instance type for k0s controller nodes."
  type        = string
  default     = "PLAY2-MICRO"
}

variable "gpu_instance_type" {
  description = "Scaleway instance type for GPU worker nodes (L4)."
  type        = string
  default     = "L4-1-24G"
}

variable "admin_ipv4_cidr" {
  description = "CIDR allowed to access SSH and Kubernetes API (if exposed publicly)."
  type        = string
  default     = "0.0.0.0/0"
}
