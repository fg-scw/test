# Example tfvars for the k0s + KubeRay + vLLM multi-AZ cluster

ssh_public_key       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDXaSVcJLERTOT4WHfDqRX/x6cCGRjnbDbeIrq4Lnrm7KJTh6vvuAzZXqVSGoCL8uyrYi8y8bwMj82NHjtT7YVTus7UL7wmEYZgCzOfPEsrGKB3EgM5+tqW4Ze8/39H2Y6/63fOTfpnmVdVHF8r/gfHgIiYDlID6m3xEGptmw96AljGkGfZ/VEoMeHLzm0T+DtDvIX6wpvtM1j2VBk+SrCT95XfugU9Xv498wVpu1kQB9ZP7kXx8FIiGDOfpcg8BK2N59f9ESDTIDyTBQNNnp2f+4Fucgwqzg8iM+eUDj2HEAp8/wdmbTy9nVPLnr3RRDWd6KoV8+xqOeCeskDXYkn4otG7KyswSO8pH+WlytRYq6lOw0V7IeLRQ4x8+u5zE/1c3mRbArywxgOnY9ziuPKILZqto+L8Dk/Zqdqn4zOBZWQ0MuPBk0e4kuKctVkMNa+2HpJ41Zs+MeTc8CM40qP5ODehVzIfz8+E5FhJZ1XOU7mncBpepd3+/leLYarTaAU= fabienganderatz@MacBook-Pro-de-Fabien-2.local"
ssh_private_key_path = "~/.ssh/id_rsa"

# Optional: override instance types / access CIDR
controller_instance_type = "PLAY2-MICRO"
gpu_instance_type        = "L4-1-24G"
# admin_ipv4_cidr          = "203.0.113.0/24"
