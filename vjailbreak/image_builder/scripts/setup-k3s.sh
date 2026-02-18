#!/bin/bash
set -euo pipefail

# K3s version to install
K3S_VERSION="${K3S_VERSION:-v1.31.4+k3s1}"

# URL-encode the version (replace + with %2B)
K3S_VERSION_URL="${K3S_VERSION//+/%2B}"

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log "Setting up K3s ${K3S_VERSION} binaries..."

# Create k3s setup directory
sudo mkdir -p /etc/pf9/k3s-setup

# Download K3s binary
log "Downloading K3s binary..."
curl -sfL "https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION_URL}/k3s" -o /tmp/k3s || { log "Failed to download K3s binary"; exit 1; }
sudo mv /tmp/k3s /etc/pf9/k3s-setup/k3s
sudo chmod +x /etc/pf9/k3s-setup/k3s

# Download K3s install script
log "Downloading K3s install script..."
curl -sfL https://get.k3s.io -o /tmp/k3s-install.sh
sudo mv /tmp/k3s-install.sh /etc/pf9/k3s-setup/k3s-install.sh
sudo chmod +x /etc/pf9/k3s-setup/k3s-install.sh

# Create symlinks for k3s binaries (needed for INSTALL_K3S_SKIP_DOWNLOAD=true)
if [[ ! -L /usr/local/bin/k3s ]]; then
  sudo ln -sf /etc/pf9/k3s-setup/k3s /usr/local/bin/k3s
fi

# Download K3s airgap images
log "Downloading K3s airgap images..."
curl -sfL "https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION_URL}/k3s-airgap-images-amd64.tar.zst" -o /tmp/k3s-airgap-images-amd64.tar.zst
sudo mkdir -p /var/lib/rancher/k3s/agent/images/
sudo mv /tmp/k3s-airgap-images-amd64.tar.zst /var/lib/rancher/k3s/agent/images/

log "K3s setup completed successfully!"
log "K3s binary: /etc/pf9/k3s-setup/k3s"
log "K3s install script: /etc/pf9/k3s-setup/k3s-install.sh"
log "K3s airgap images: /var/lib/rancher/k3s/agent/images/"
