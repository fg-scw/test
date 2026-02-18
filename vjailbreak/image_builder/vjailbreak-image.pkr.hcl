packer {
  required_plugins {
    qemu = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

variable "ubuntu_minimal_url" {
  type    = string
  default = "https://cloud-images.ubuntu.com/minimal/releases/jammy/release/ubuntu-22.04-minimal-cloudimg-amd64.img"
}

variable "ubuntu_minimal_checksum" {
  type    = string
  default = "file:https://cloud-images.ubuntu.com/minimal/releases/jammy/release/SHA256SUMS"
}

variable "k3s_version" {
  type    = string
  default = "v1.31.4+k3s1"
}

variable "helm_version" {
  type    = string
  default = "v3.16.3"
}

source "qemu" "vjailbreak-image" {
  disk_image           = true
  skip_compaction      = true
  iso_url              = var.ubuntu_minimal_url
  iso_checksum         = var.ubuntu_minimal_checksum
  iso_target_extension = "qcow2"
  output_directory     = "vjailbreak_qcow2"
  vm_name              = "vjailbreak-image.qcow2"
  disk_size            = "10G"
  format               = "qcow2"
  headless             = true
  accelerator          = "kvm"
  ssh_username         = "ubuntu"
  ssh_password         = "password"
  ssh_timeout          = "20m"
  cpus                 = 2
  memory               = 2048
  efi_boot             = false
  shutdown_command     = "echo 'password' | sudo -S shutdown -P now"
  boot_wait            = "10s"

  http_directory = "${path.root}/cloudinit/"

  qemuargs = [
    ["-smbios", "type=1,serial=ds=nocloud-net;s=http://{{ .HTTPIP }}:{{ .HTTPPort }}/"]
  ]
}

build {
  sources = ["source.qemu.vjailbreak-image"]

  provisioner "file" {
    source      = "${path.root}/deploy"
    destination = "/tmp/yamls"
  }

  provisioner "file" {
    source      = "${path.root}/scripts/install.sh"
    destination = "/tmp/install.sh"
  }
  provisioner "file" {
    source      = "${path.root}/scripts/setup-k3s.sh"
    destination = "/tmp/setup-k3s.sh"
  }
  provisioner "file" {
    source      = "${path.root}/scripts/pf9-htpasswd.sh"
    destination = "/tmp/pf9-htpasswd.sh"
  }

  provisioner "file" {
    source      = "${path.root}/scripts/log_collector.sh"
    destination = "/tmp/log_collector.sh"
  }
  provisioner "file" {
    source      = "${path.root}/scripts/user_setup_daemon.sh"
    destination = "/tmp/user_setup_daemon.sh"
  }
  provisioner "file" {
    source      = "${path.root}/configs/k3s.env"
    destination = "/tmp/k3s.env"
  }

  provisioner "file" {
    source      = "${path.root}/configs/daemonset.yaml"
    destination = "/tmp/daemonset.yaml"
  }

  provisioner "file" {
    source      = "${path.root}/configs/rsyncd.conf"
    destination = "/tmp/rsyncd.conf"
  }
  
  provisioner "file" {
    source      = "${path.root}/configs/env"
    destination = "/tmp/env"
  }

  provisioner "file" {
    source      = "${path.root}/configs/vjailbreak-settings.yaml"
    destination = "/tmp/vjailbreak-settings.yaml"
  }

  provisioner "file" {
    source      = "${path.root}/cronjob/version-checker.yaml"
    destination = "/tmp/version-checker.yaml"
  }

  provisioner "file" {
    source      = "${path.root}/images"
    destination = "/home/ubuntu"
  }

  provisioner "file" {
    source      = "${path.root}/opensource.txt"
    destination = "/tmp/opensource.txt"
  }
  provisioner "shell" {
    environment_vars = [
      "K3S_VERSION=${var.k3s_version}"
    ]
    inline = [
    "sudo mkdir -p /etc/pf9",
    "chmod +x /tmp/setup-k3s.sh",
    "sudo -E /tmp/setup-k3s.sh",
    "sudo mv /tmp/install.sh /etc/pf9/install.sh",
    "sudo mv /tmp/pf9-htpasswd.sh /etc/pf9/pf9-htpasswd.sh",
    "sudo mv /tmp/log_collector.sh /etc/pf9/log_collector.sh",
    "sudo mv /tmp/k3s.env /etc/pf9/k3s.env",
    "sudo mkdir -p /etc/pf9/images",
    "sudo mv /home/ubuntu/images/* /etc/pf9/images",
    "sudo mkdir -p /home/ubuntu/virtio-win",
    "sudo chown -R ubuntu:ubuntu /home/ubuntu/virtio-win",
    "sudo mv /etc/pf9/images/virtio-win.iso /home/ubuntu/virtio-win/virtio-win.iso",
    "sudo mv /tmp/yamls /etc/pf9/yamls",
    "sudo mv /tmp/rsyncd.conf /etc/pf9/rsyncd.conf",
    "sudo mv /tmp/daemonset.yaml /etc/pf9/yamls/daemonset.yaml",
    "sudo mv /tmp/env /etc/pf9/env",
    "sudo mv /tmp/vjailbreak-settings.yaml /etc/pf9/yamls/vjailbreak-settings.yaml",
    "sudo mv /tmp/version-checker.yaml /etc/pf9/yamls/version-checker.yaml",
    "sudo mv /tmp/opensource.txt /home/ubuntu/opensource.txt",
    "sudo chmod +x /etc/pf9/install.sh",
    "sudo chmod +r /etc/pf9/pf9-htpasswd.sh",
    "sudo chmod +x /etc/pf9/log_collector.sh",
    "sudo chown root:root /etc/pf9/k3s.env",
    "sudo chmod 644 /etc/pf9/k3s.env",
    "sudo chmod 644 /etc/pf9/env",
    "sudo chmod +x /tmp/user_setup_daemon.sh",
    "sudo df -h",
    "sudo bash /tmp/user_setup_daemon.sh",
    "sudo apt-get update",
    "sudo apt-get install -y --no-install-recommends cron curl ca-certificates python3-openstackclient",
    "sudo systemctl enable cron",
    "echo '@reboot root /etc/pf9/install.sh' | sudo tee -a /etc/crontab",
    "curl -fsSL -o /tmp/get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3",
    "chmod 700 /tmp/get_helm.sh",
    "DESIRED_VERSION=${var.helm_version} /tmp/get_helm.sh",
    "rm /tmp/get_helm.sh",
    "sudo helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx",
    "sudo helm repo update",
    "sudo helm pull ingress-nginx/ingress-nginx --untar --untardir /etc/pf9/",
    "sudo apt-get clean",
    "sudo rm -rf /var/lib/apt/lists/*",
    ]
  }
}

