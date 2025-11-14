# Guide de déploiement Talos Kubernetes sur Scaleway

Ce guide détaille le processus complet de déploiement d'un cluster Kubernetes hautement disponible avec Talos Linux sur Scaleway.

## 🎯 Vue d'ensemble

Ce déploiement créera :
- **1 VPC régional** avec Private Network IPAM automatique
- **3 Control Planes** distribués sur PAR-1, PAR-2, PAR-3
- **3 Workers** (ou plus) distribués multi-AZ
- **1 Load Balancer** interne pour l'API Kubernetes
- **1 Public Gateway** pour NAT et bastion SSH
- **Cilium CNI** avec kube-proxy replacement

## 📋 Prérequis

### Compte Scaleway

1. Créez un compte sur https://console.scaleway.com
2. Créez un projet ou utilisez le projet par défaut
3. Générez des credentials API :
   - Organization > Credentials
   - Créer un token API
   - Notez l'Access Key et la Secret Key

### Outils locaux

```bash
# Terraform
brew install terraform  # macOS
# ou téléchargez depuis https://www.terraform.io/downloads

# Packer
brew install packer  # macOS
# ou téléchargez depuis https://www.packer.io/downloads

# talosctl (version correspondant à Talos)
curl -sL https://talos.dev/install | sh

# kubectl
brew install kubectl  # macOS

# helm (optionnel)
brew install helm  # macOS

# scw CLI (optionnel mais recommandé)
brew install scw  # macOS
scw init  # Configurer avec vos credentials
```

## 🚀 Étape 1 : Configuration initiale

### Cloner le repository

```bash
git clone https://github.com/votrecompte/talos-scaleway.git
cd talos-scaleway
```

### Configurer les credentials

```bash
# Copier le template
cp .envrc.sample .envrc

# Éditer avec vos credentials
vim .envrc

# Charger les variables
source .envrc
```

Contenu de `.envrc` :

```bash
export SCW_ACCESS_KEY="SCWXXXXXXXXXXXXXXXXX"
export SCW_SECRET_KEY="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export SCW_DEFAULT_PROJECT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export SCW_DEFAULT_REGION="fr-par"
export SCW_DEFAULT_ZONE="fr-par-1"

export TF_VAR_scw_access_key="$SCW_ACCESS_KEY"
export TF_VAR_scw_secret_key="$SCW_SECRET_KEY"
export TF_VAR_scw_project_id="$SCW_DEFAULT_PROJECT_ID"

export PACKER_LOG=1
export PACKER_LOG_PATH="./packer.log"
```

## 🖼️ Étape 2 : Création de l'image Talos

### Processus de build Packer

Packer va :
1. Créer une instance temporaire Ubuntu
2. Télécharger l'image Talos depuis Image Factory
3. Écrire l'image sur le disque de l'instance
4. Créer un snapshot
5. Générer une image réutilisable
6. Nettoyer les ressources temporaires

### Exécution du build

```bash
cd packer

# Initialiser Packer
packer init .

# Valider la configuration
packer validate -var="talos_version=v1.11.5" .

# Build de l'image
packer build -var="talos_version=v1.11.5" .
```

**Durée** : 5-10 minutes

**Output** : Une image nommée `talos-scaleway-v1.11.5-YYYYMMDD-HHMMSS`

### Vérification

```bash
# Lister les images créées
scw instance image list | grep talos

# Ou via Terraform
data "scaleway_instance_image" "talos" {
  name         = "talos-scaleway-v1.11.5"
  architecture = "x86_64"
  latest       = true
}
```

## 🏗️ Étape 3 : Déploiement de l'infrastructure

### Configuration Terraform

```bash
cd ../terraform

# Copier le fichier d'exemple
cp terraform.tfvars.example terraform.tfvars

# Éditer avec vos paramètres
vim terraform.tfvars
```

**Configuration minimale** :

```hcl
cluster_name  = "talos-prod"
environment   = "production"
talos_version = "v1.11.5"

# Multi-AZ
enable_multi_az = true

# Control Planes
control_plane_count         = 3
control_plane_instance_type = "PRO2-S"
control_plane_disk_size     = 50

# Workers
worker_count         = 3
worker_instance_type = "PRO2-M"
worker_disk_size     = 100

# Load Balancer privé
expose_k8s_api_publicly = false

# Public Gateway avec bastion
enable_bastion_on_gateway = true
```

### Déploiement

```bash
# Initialiser Terraform
terraform init

# Planifier les changements
terraform plan

# Vérifier le plan (ressources créées, coûts, etc.)

# Appliquer
terraform apply
```

**Durée** : 5-10 minutes

### Vérifier les outputs

```bash
# Endpoint API Kubernetes
terraform output kubernetes_api_endpoint

# IPs des nœuds
terraform output control_plane_ips
terraform output worker_ips

# Distribution multi-AZ
terraform output control_plane_distribution
terraform output workers_distribution

# IP de la Public Gateway (NAT)
terraform output public_gateway_ip
```

## ⚙️ Étape 4 : Configuration Talos

### Générer les configurations

```bash
# Récupérer l'endpoint API
K8S_API=$(terraform output -raw kubernetes_api_endpoint)

# Générer les configurations
talosctl gen config talos-prod $K8S_API \
  --output-dir _out \
  --with-docs=false \
  --with-examples=false
```

**Fichiers générés** :
- `_out/talosconfig` : Configuration pour talosctl
- `_out/controlplane.yaml` : Configuration des control planes
- `_out/worker.yaml` : Configuration des workers

### Patcher pour Cilium

```bash
# Appliquer le patch pour désactiver kube-proxy
talosctl --talosconfig _out/talosconfig machineconfig patch \
  _out/controlplane.yaml \
  --patch @../cilium-patch.yaml \
  -o _out/controlplane-patched.yaml
```

### Appliquer les configurations

```bash
# Récupérer les IPs
CONTROL_PLANE_IPS=$(terraform output -json control_plane_ips | jq -r '.[]')
WORKER_IPS=$(terraform output -json worker_ips | jq -r '.[]')

# Appliquer aux control planes
for ip in $CONTROL_PLANE_IPS; do
  echo "Applying config to control plane: $ip"
  talosctl apply-config --insecure \
    --nodes $ip \
    --file _out/controlplane-patched.yaml
done

# Attendre 2-3 minutes que les control planes démarrent

# Appliquer aux workers
for ip in $WORKER_IPS; do
  echo "Applying config to worker: $ip"
  talosctl apply-config --insecure \
    --nodes $ip \
    --file _out/worker.yaml
done
```

### Bootstrap etcd

```bash
# Récupérer le premier control plane
FIRST_CP=$(terraform output -json control_plane_ips | jq -r '.[0]')

# Bootstrap etcd (une seule fois!)
talosctl --talosconfig _out/talosconfig bootstrap --nodes $FIRST_CP
```

**⚠️ IMPORTANT** : Le bootstrap ne doit être fait qu'une seule fois sur un seul control plane !

### Vérifier le cluster

```bash
# Récupérer le kubeconfig
K8S_API_IP=$(terraform output -raw kubernetes_api_ip)
talosctl --talosconfig _out/talosconfig \
  kubeconfig _out/kubeconfig \
  --nodes $K8S_API_IP

# Configurer kubectl
export KUBECONFIG=$(pwd)/_out/kubeconfig

# Vérifier les nœuds (ils seront NotReady sans CNI)
kubectl get nodes
```

## 🌐 Étape 5 : Installation Cilium

### Ajouter le repo Helm

```bash
helm repo add cilium https://helm.cilium.io/
helm repo update
```

### Installer Cilium

```bash
K8S_API_IP=$(terraform output -raw kubernetes_api_ip)

helm install cilium cilium/cilium \
  --namespace kube-system \
  --set ipam.mode=kubernetes \
  --set kubeProxyReplacement=true \
  --set k8sServiceHost=$K8S_API_IP \
  --set k8sServicePort=6443 \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true
```

### Vérifier l'installation

```bash
# Attendre que les pods Cilium démarrent
kubectl -n kube-system get pods -l k8s-app=cilium -w

# Vérifier le status Cilium
cilium status --wait

# Les nœuds devraient maintenant être Ready
kubectl get nodes

# Vérifier la connectivité
cilium connectivity test
```

## ✅ Étape 6 : Tests et validation

### Test de déploiement

```bash
# Déployer nginx de test
kubectl create deployment nginx --image=nginx --replicas=3

# Exposer via NodePort
kubectl expose deployment nginx --type=NodePort --port=80

# Vérifier
kubectl get pods -o wide
kubectl get svc nginx
```

### Test de connectivité réseau

```bash
# Depuis un pod vers l'extérieur
kubectl run -it --rm debug --image=nicolaka/netshoot --restart=Never -- curl -I https://www.scaleway.com

# Entre pods
kubectl run -it --rm debug --image=nicolaka/netshoot --restart=Never -- curl <NGINX_POD_IP>
```

### Test multi-AZ

```bash
# Vérifier la distribution des pods
kubectl get pods -o wide | grep nginx

# Simuler une panne de zone (drain)
kubectl drain <NODE_NAME> --ignore-daemonsets --delete-emptydir-data

# Vérifier que les pods sont reschedulés
kubectl get pods -o wide
```

## 🔧 Maintenance et opérations

### Accès via bastion SSH

```bash
# Via Public Gateway bastion
GATEWAY_IP=$(terraform output -raw public_gateway_ip)
ssh -J bastion@$GATEWAY_IP:61000 root@<CONTROL_PLANE_IP>
```

### Mise à jour de Talos

```bash
# Générer nouvelle image avec Packer
cd packer
packer build -var="talos_version=v1.12.0" .

# Mettre à jour Terraform
cd ../terraform
vim terraform.tfvars  # Changer talos_version

# Appliquer (Terraform recréera les instances)
terraform apply
```

### Scaling des workers

```bash
# Éditer terraform.tfvars
vim terraform.tfvars  # Changer worker_count

# Appliquer
terraform apply

# Appliquer config aux nouveaux workers
# (voir Étape 4)
```

### Backup etcd

```bash
# Via talosctl
talosctl -n <CONTROL_PLANE_IP> etcd snapshot

# Télécharger le snapshot
talosctl -n <CONTROL_PLANE_IP> cp /var/lib/etcd/snapshots/etcd.snapshot ./
```

## 🧹 Nettoyage

### Supprimer le cluster

```bash
cd terraform
terraform destroy
```

### Supprimer l'image Packer

```bash
# Lister les images
scw instance image list | grep talos

# Supprimer
scw instance image delete <IMAGE_ID>
```

## 📊 Coûts estimés

**Configuration standard** (3 CP + 3 Workers, fr-par, nov 2025) :

| Ressource | Type | Quantité | Prix unitaire | Total/mois |
|-----------|------|----------|---------------|------------|
| Control Planes | PRO2-S | 3 | 0.055€/h | ~119€ |
| Workers | PRO2-M | 3 | 0.165€/h | ~356€ |
| Block Storage | SBS 5K | 450 GB | 0.088€/GB | ~40€ |
| Load Balancer | LB-S | 1 | 0.02€/h | ~14€ |
| Public Gateway | VPC-GW-S | 1 | 0.02€/h | ~14€ |
| **TOTAL** | | | | **~543€/mois** |

**Configuration développement** (1 CP + 2 Workers) : ~200€/mois

## 🆘 Troubleshooting

### Les nœuds ne démarrent pas

```bash
# Vérifier les logs Talos
talosctl -n <NODE_IP> logs

# Vérifier le service kubelet
talosctl -n <NODE_IP> service kubelet status

# Vérifier etcd
talosctl -n <CONTROL_PLANE_IP> service etcd status
```

### Problèmes réseau

```bash
# Vérifier la connectivité au Private Network
ping <PRIVATE_IP>

# Vérifier la Public Gateway
curl -I https://www.scaleway.com  # Depuis un nœud

# Vérifier les routes
ip route show
```

### Cilium ne démarre pas

```bash
# Logs des pods Cilium
kubectl -n kube-system logs -l k8s-app=cilium

# Status détaillé
cilium status

# Restart des pods
kubectl -n kube-system delete pods -l k8s-app=cilium
```

## 📚 Ressources

- **Talos Linux** : https://www.talos.dev/
- **Scaleway Docs** : https://www.scaleway.com/en/docs/
- **Cilium** : https://docs.cilium.io/
- **Terraform Scaleway** : https://registry.terraform.io/providers/scaleway/scaleway/

## 🤝 Support

Pour des questions ou problèmes :
- GitHub Issues : https://github.com/votrecompte/talos-scaleway/issues
- Scaleway Community : https://slack.scaleway.com/
- Talos Community : https://www.talos.dev/community/
