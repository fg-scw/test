#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
  echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
  echo -e "${RED}[ERROR]${NC} $1"
  exit 1
}

K0SCTL_VERSION="0.16.0"
K0SCTL_BIN="./k0sctl"

download_k0sctl() {
  if [[ -x "${K0SCTL_BIN}" ]]; then
    return
  fi

  log "Téléchargement de k0sctl ${K0SCTL_VERSION}..."

  OS=$(uname | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)
  case "${ARCH}" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
  esac

  URL="https://github.com/k0sproject/k0sctl/releases/download/v${K0SCTL_VERSION}/k0sctl-${OS}-${ARCH}"

  curl -fsSL "${URL}" -o "${K0SCTL_BIN}" || error "Impossible de télécharger k0sctl depuis ${URL}"
  chmod +x "${K0SCTL_BIN}"
}

check_prerequisites() {
  log "Vérification des prérequis..."

  command -v terraform >/dev/null 2>&1 || error "terraform n'est pas installé"
  command -v kubectl   >/dev/null 2>&1 || error "kubectl n'est pas installé"
  command -v helm      >/dev/null 2>&1 || error "helm n'est pas installé"
  command -v jq        >/dev/null 2>&1 || error "jq n'est pas installé"
  command -v curl      >/dev/null 2>&1 || error "curl n'est pas installé"

  download_k0sctl
}

deploy_infrastructure() {
  log "Déploiement de l'infrastructure Scaleway (multi-AZ + GPU)..."
  terraform init
  terraform apply -auto-approve

  log "Génération de k0sctl.yaml depuis les outputs Terraform..."
  terraform output -raw k0sctl_config > k0sctl.yaml
}

deploy_kubernetes() {
  log "Installation / mise à jour du cluster k0s..."
  ${K0SCTL_BIN} apply --config k0sctl.yaml

  log "Récupération du kubeconfig..."
  ${K0SCTL_BIN} kubeconfig --config k0sctl.yaml > kubeconfig
  export KUBECONFIG="$(pwd)/kubeconfig"

  log "Attente que tous les noeuds Kubernetes soient prêts..."
  kubectl wait --for=condition=Ready nodes --all --timeout=900s || {
    warn "Les noeuds ne sont pas tous Ready, affichage de l'état courant..."
    kubectl get nodes -o wide || true
    kubectl describe nodes || true
    error "Les noeuds ne sont pas prêts"
  }
}

add_topology_labels() {
  export KUBECONFIG="$(pwd)/kubeconfig"

  log "Ajout des labels topology.kubernetes.io/zone sur tous les nœuds..."

  kubectl label node k0s-controller-fr-par-1 topology.kubernetes.io/zone=fr-par-1 topology.kubernetes.io/region=fr-par --overwrite
  kubectl label node k0s-controller-fr-par-2 topology.kubernetes.io/zone=fr-par-2 topology.kubernetes.io/region=fr-par --overwrite
  kubectl label node k0s-gpu-fr-par-1 topology.kubernetes.io/zone=fr-par-1 topology.kubernetes.io/region=fr-par --overwrite
  kubectl label node k0s-gpu-fr-par-2 topology.kubernetes.io/zone=fr-par-2 topology.kubernetes.io/region=fr-par --overwrite

  log "Labels topology ajoutés avec succès"
  kubectl get nodes -L topology.kubernetes.io/zone,topology.kubernetes.io/region
}


install_nvidia_plugin() {
  export KUBECONFIG="$(pwd)/kubeconfig"

  log "Installation du plugin NVIDIA (Helm, chart local)..."

  if [[ ! -d "nvidia-device-plugin" ]]; then
    error "Le répertoire ./nvidia-device-plugin n'existe pas. Place le chart ici (à côté de deploy.sh) ou adapte le chemin dans deploy.sh."
  fi

  log "Mise à jour des dépendances Helm du plugin NVIDIA..."
  helm dependency update ./nvidia-device-plugin >/dev/null || \
    log "Impossible de mettre à jour les dépendances Helm (charts/). Je continue quand même."

  log "Déploiement / mise à jour du chart nvidia-device-plugin..."
  helm upgrade --install nvidia-device-plugin ./nvidia-device-plugin \
    --namespace kube-system \
    --create-namespace \
    --set image.repository=nvcr.io/nvidia/k8s-device-plugin \
    --set image.tag=v0.18.0 \
    --set gfd.enabled=true \
    --set nfd.enabled=true

  # Wait for daemonsets to be created
  sleep 10

  # Le DaemonSet s'appelle "nvidia-device-plugin" (cf. kubectl get ds)
  local ds_name="nvidia-device-plugin"

  if ! kubectl -n kube-system get ds "${ds_name}" >/dev/null 2>&1; then
    error "DaemonSet ${ds_name} introuvable dans kube-system (regarde kubectl -n kube-system get ds)."
  fi

  log "Attente de la disponibilité du DaemonSet NVIDIA (${ds_name})..."
  # Give it more time and don't fail immediately
  kubectl -n kube-system rollout status "ds/${ds_name}" --timeout=600s || {
    warn "Le DaemonSet ${ds_name} n'est pas complètement prêt. Vérification des pods..."
    kubectl get pods -n kube-system -l app.kubernetes.io/name=nvidia-device-plugin -o wide
    
    # Check if at least some pods are running
    running_pods=$(kubectl get pods -n kube-system -l app.kubernetes.io/name=nvidia-device-plugin -o json | jq '[.items[] | select(.status.phase == "Running")] | length')
    if [[ ${running_pods} -gt 0 ]]; then
      warn "Certains pods NVIDIA sont en cours de démarrage, mais ${running_pods} sont déjà Running. Je continue..."
    else
      log "Logs des pods NVIDIA en erreur:"
      kubectl logs -n kube-system -l app.kubernetes.io/name=nvidia-device-plugin --tail=20 --prefix=true || true
      error "Aucun pod NVIDIA n'est Running. Vérifiez les logs ci-dessus."
    fi
  }

  # Verify GPU allocation
  log "Vérification de l'exposition des GPUs..."
  kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU-CAPACITY:.status.capacity.'nvidia\.com/gpu',GPU-ALLOCATABLE:.status.allocatable.'nvidia\.com/gpu'
}

label_nodes_for_kuberay() {
  export KUBECONFIG="$(pwd)/kubeconfig"

  log "Labellisation des nœuds pour KubeRay / vLLM..."

  # Head Ray sur les contrôleurs
  kubectl label node k0s-controller-fr-par-1 ray-node-type=head --overwrite
  kubectl label node k0s-controller-fr-par-2 ray-node-type=head --overwrite

  # Workers Ray sur les nœuds GPU
  kubectl label node k0s-gpu-fr-par-1 ray-node-type=worker --overwrite
  kubectl label node k0s-gpu-fr-par-2 ray-node-type=worker --overwrite

  # Label GPU attendu par le RayService (en plus du label feature.* déjà présent)
  kubectl label node k0s-gpu-fr-par-1 nvidia.com/gpu.present=true --overwrite
  kubectl label node k0s-gpu-fr-par-2 nvidia.com/gpu.present=true --overwrite
}

install_kuberay() {
  export KUBECONFIG="$(pwd)/kubeconfig"

  log "Installation de KubeRay (operator)..."
  helm repo add kuberay https://ray-project.github.io/kuberay-helm/ >/dev/null
  helm repo update kuberay >/dev/null

  helm upgrade --install kuberay-operator kuberay/kuberay-operator     --namespace ray-system     --create-namespace

  log "Attente que l'operator KubeRay soit prêt..."
  kubectl -n ray-system rollout status deploy/kuberay-operator --timeout=600s
}

deploy_vllm_service() {
  export KUBECONFIG="$(pwd)/kubeconfig"

  log "Déploiement du RayService vLLM multi-AZ..."
  kubectl apply -f vllm-service.yaml

  log "Attente de la création des ressources Ray..."
  sleep 30
}

show_status() {
  export KUBECONFIG="$(pwd)/kubeconfig"

  log "=== État du cluster k0s ==="
  kubectl get nodes -o wide -L topology.kubernetes.io/zone,ray-node-type,nvidia.com/gpu.present

  log "=== DaemonSets NVIDIA ==="
  kubectl get ds -n kube-system | grep nvidia || log "Aucun DaemonSet NVIDIA trouvé"

  log "=== Pods NVIDIA ==="
  kubectl get pods -n kube-system -l app.kubernetes.io/name=nvidia-device-plugin -o wide

  log "=== Exposition GPU par nœud ==="
  kubectl get nodes -o custom-columns=NAME:.metadata.name,ZONE:.metadata.labels.'topology\.kubernetes\.io/zone',GPU-CAPACITY:.status.capacity.'nvidia\.com/gpu',GPU-ALLOCATABLE:.status.allocatable.'nvidia\.com/gpu'

  log "=== Pods Ray / vLLM ==="
  kubectl get pods -A | grep -E "ray|vllm" || log "Aucun pod Ray/vLLM trouvé"

  log "=== Ressources RayService ==="
  kubectl get rayservices.ray.io -A || log "Aucun RayService trouvé"

  log ""
  log "Pour tester l'API vLLM, expose un endpoint (port-forward / ingress) et utilise:"
  log "  kubectl port-forward -n default svc/vllm-federated-service-mjv2f-head-svc 8000:8000"
  log "  ./test-vllm.sh http://localhost:8000/v1"
  log ""
  log "Si des problèmes persistent:"
  log "  - Vérifiez les logs NVIDIA: kubectl logs -n kube-system <nvidia-pod-name>"
  log "  - Vérifiez nvidia-smi sur les GPU nodes: ssh root@<gpu-ip> nvidia-smi"
  log "  - Vérifiez les labels: kubectl get nodes --show-labels"
}

main() {
  log "=== Déploiement vLLM multi-AZ sur Scaleway (k0s + KubeRay) ==="

  check_prerequisites
  deploy_infrastructure
  deploy_kubernetes
  add_topology_labels
  install_nvidia_plugin
  label_nodes_for_kuberay
  install_kuberay
  deploy_vllm_service
  show_status

  log "=== Déploiement terminé avec succès ==="
}

main
