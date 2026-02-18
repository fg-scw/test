#!/bin/bash
# This script downloads the ingress-nginx controller and kube-webhook-certgen images and exports them as tar files.
# But this script has been run already in base image, so this is just a placeholder for now.
# In case in the future we need to download images, we can use this script.
set -euo pipefail


# get tag from /etc/pf9/yamls/01ui.yaml
TAG=$1

REGISTRY="quay.io"
REPO="platform9"
kube_state_metrics="registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.13.0"
prometheus_adapter="registry.k8s.io/prometheus-adapter/prometheus-adapter:v0.12.0"
prometheus="quay.io/prometheus/prometheus:v2.54.1"
alertmanager="quay.io/prometheus/alertmanager:v0.27.0"
blackbox_exporter="quay.io/prometheus/blackbox-exporter:v0.25.0"
node_exporter="quay.io/prometheus/node-exporter:v1.8.2"
pushgateway="quay.io/prometheus/pushgateway:v1.5.0"
kube_rbac_proxy="quay.io/brancz/kube-rbac-proxy:v0.19.1"
prometheus_config_reloader="quay.io/prometheus-operator/prometheus-config-reloader:v0.76.0"
prometheus_operator="quay.io/prometheus-operator/prometheus-operator:v0.76.0"
configmap_reload="ghcr.io/jimmidyson/configmap-reload:v0.13.1"
grafana="docker.io/grafana/grafana:12.3.2"
v2v_helper="quay.io/platform9/vjailbreak-v2v-helper:$TAG"
controller="quay.io/platform9/vjailbreak-controller:$TAG"
ui="quay.io/platform9/vjailbreak-ui:$TAG"
vpwned="quay.io/platform9/vjailbreak-vpwned:$TAG"
virtiowin="https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso"
# TODO(suhas): Create a seperate repository for alpine image in quay
alpine="quay.io/platform9/vjailbreak:alpine"

CERT_MANAGER_VERSION="v1.16.1"
cert_manager_controller="quay.io/jetstack/cert-manager-controller:${CERT_MANAGER_VERSION}"
cert_manager_webhook="quay.io/jetstack/cert-manager-webhook:${CERT_MANAGER_VERSION}"
cert_manager_cainjector="quay.io/jetstack/cert-manager-cainjector:${CERT_MANAGER_VERSION}"

# Download cert-manager manifests
CERT_MANAGER_URL="https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
MANIFEST_PATH="image_builder/cert-manager-manifests/cert-manager.yaml"
mkdir -p image_builder/cert-manager-manifests
curl -L "${CERT_MANAGER_URL}" -o "${MANIFEST_PATH}"

sed -i 's/imagePullPolicy: Always/imagePullPolicy: IfNotPresent/g' "${MANIFEST_PATH}"


# Download and export images
images=(
  "$kube_state_metrics"
  "$prometheus_adapter"
  "$prometheus"
  "$alertmanager"
  "$blackbox_exporter"
  "$node_exporter"
  "$pushgateway"
  "$prometheus_config_reloader"
  "$prometheus_operator"
  "$v2v_helper"
  "$controller"
  "$ui"
  "$configmap_reload"
  "$grafana"
  "$alpine"
  "$vpwned"
  "$cert_manager_controller"
  "$cert_manager_webhook"
  "$cert_manager_cainjector"
)

for img in "${images[@]}"; do
  echo "[*] Pulling $img"
  sudo ctr  i pull "$img"

  tag=$(echo "$img" | cut -d'@' -f1)
  fname=$(echo "$tag" | tr '/:@' '_')

  echo "[*] Exporting to $fname.tar"
  sudo ctr i export "image_builder/images/$fname.tar" "$img"
done


ctr images pull --all-platforms quay.io/brancz/kube-rbac-proxy:v0.19.1
sleep 10
ctr images export "image_builder/images/kube-rbac-proxy.tar" quay.io/brancz/kube-rbac-proxy:v0.19.1

echo "[✔] All images downloaded and exported as tar files."


# Download virtio-win.iso
echo "[*] Downloading virtio-win.iso"
wget -O image_builder/images/virtio-win.iso "$virtiowin"
