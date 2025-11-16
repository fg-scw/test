# Talos Kubernetes sur Scaleway

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Talos](https://img.shields.io/badge/Talos-v1.11.5-blue.svg)](https://www.talos.dev/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.31-blue.svg)](https://kubernetes.io/)

Déploiement automatisé d'un cluster Kubernetes hautement disponible avec **Talos Linux** sur le cloud **Scaleway**.

Ce projet construit une infrastructure complète pour exécuter Kubernetes sur Scaleway avec :

- **Talos Linux** : OS immuable et API-driven pour Kubernetes
- **Terraform** : Infrastructure as Code pour le provisioning
- **Packer** : Automatisation de la création d'images personnalisées
- **Cilium** : CNI moderne basé sur eBPF avec kube-proxy replacement
- **VPC & Private Networks** : Réseau isolé avec IPAM automatique

## 📋 Table des matières

* [🏗 Architecture](#-architecture)
* [🔧 Prérequis](#-prérequis)
* [🚀 Démarrage rapide](#-démarrage-rapide)
* [📁 Structure du projet](#-structure-du-projet)
* [📚 Documentation](#-documentation)
* [🤝 Contribution](#-contribution)
* [📝 Licence](#-licence)

## 🏗 Architecture

### Topologie réseau

L'infrastructure repose sur une architecture multi-AZ hautement disponible avec VPC Scaleway :

```
┌──────────────────────────────────────────────────────────────────────┐
│                    Scaleway Cloud (fr-par)                          │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ VPC Kubernetes (Regional)                                      │  │
│  │                                                                │  │
│  │  ┌──────────────────────────────────────────────────────────┐  │  │
│  │  │ Private Network (10.0.0.0/22) - IPAM automatique         │  │  │
│  │  │                                                          │  │  │
│  │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │  │
│  │  │  │ PAR-1        │  │ PAR-2        │  │ PAR-3        │  │  │  │
│  │  │  ├──────────────┤  ├──────────────┤  ├──────────────┤  │  │  │
│  │  │  │ CP-1         │  │ CP-2         │  │ CP-3         │  │  │  │
│  │  │  │ Worker-1     │  │ Worker-2     │  │ Worker-3     │  │  │  │
│  │  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │  │
│  │  │                                                          │  │  │
│  │  └──────────────────────────────────────────────────────────┘  │  │
│  │                                                                │  │
│  │  Load Balancer (Internal) - API Kubernetes                     │  │
│  │  ├─> Control-Plane-1:6443 (PAR-1)                              │  │
│  │  ├─> Control-Plane-2:6443 (PAR-2)                              │  │
│  │  └─> Control-Plane-3:6443 (PAR-3)                              │  │
│  │                                                                │  │
│  │  Public Gateway - NAT pour accès internet                      │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Bastion (VPC séparé) - Point d'entrée administration              │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### Composants

**Réseau Scaleway**

- VPC régional (fr-par)
- Private Network avec IPAM automatique (/22 = 1020 IPs)
- Public Gateway pour NAT (accès internet sortant)
- Load Balancer interne pour API Kubernetes
- DNS interne automatique (resources.pn.internal)

**CNI & Réseau Kubernetes**

- CNI : Cilium (eBPF, native routing mode)
- Pod CIDR : `10.244.0.0/16`
- Service CIDR : `10.96.0.0/12`
- kube-proxy : Désactivé (remplacé par Cilium)
- MTU : 1500 (default Scaleway Private Network)

**Control Plane (3 nœuds)**

- Distribution : 3 zones de disponibilité (PAR-1, PAR-2, PAR-3)
- Type : `PRO2-S` (2 vCPU, 8 GB RAM) ou `DEV1-M` (test)
- Rôles : etcd, kube-apiserver, kube-controller-manager, kube-scheduler
- HA : Quorum etcd 3 nœuds (tolère 1 panne)
- Stockage : SBS 5K, 50 GB minimum

**Workers (3+ nœuds)**

- Distribution : 3 zones de disponibilité minimum
- Type : Configurable selon les charges (PRO2-M, PRO2-L recommandés)
- Rôles : Hébergement des workloads Kubernetes
- Stockage : SBS 5K, 100 GB recommandé

**Stockage**

- Type : Block Storage SBS (NVMe)
- SBS 5K : 5000 IOPS garantis (production standard)
- SBS 15K : 15000 IOPS garantis (haute performance)
- Persistence : Volumes indépendants des instances

## 🔧 Prérequis

### Outils requis

Sur votre poste de travail :

```bash
# Terraform (>= 1.9)
terraform version

# Packer (>= 1.11)
packer version

# talosctl (version correspondant à Talos)
talosctl version

# kubectl
kubectl version --client

# helm (optionnel, pour Cilium)
helm version

# scw CLI (Scaleway CLI)
scw version
```

### ⚠️ Important : Types de volumes Scaleway

Ce projet utilise les types de volumes appropriés pour chaque cas :
- **Packer** : `l_ssd` (Local SSD pour instances temporaires)
- **Production** : `sbs_5k` (Block Storage NVMe 5K IOPS)
- **Bastion** : `l_ssd` (Local SSD suffit)

Voir [VOLUME_TYPES.md](VOLUME_TYPES.md) pour plus de détails.

### Credentials Scaleway

Vous aurez besoin de :

- Un projet Scaleway
- Accès API (Secret Key et Access Key)
- Permissions pour :
  - Créer des VPC et Private Networks
  - Créer des instances et volumes
  - Créer des images et snapshots
  - Gérer les Load Balancers et Public Gateways
  - Accès Object Storage (pour import d'images)

### Versions testées

| Composant | Version |
|-----------|---------|
| Talos Linux | v1.11.5 |
| Kubernetes | v1.31.1 |
| Cilium | v1.16+ |
| Terraform | v1.9+ |
| Packer | v1.11+ |
| Provider Scaleway | v2.62+ |

## 🚀 Démarrage rapide

### 1. Configuration des credentials

Clonez le repository et configurez vos credentials :

```bash
git clone https://github.com/votrecompte/talos-scaleway.git
cd talos-scaleway

# Copier le fichier d'exemple
cp .envrc.sample .envrc

# Éditer avec vos credentials
vim .envrc
```

Contenu de `.envrc` :

```bash
export SCW_ACCESS_KEY="VOTRE_ACCESS_KEY"
export SCW_SECRET_KEY="VOTRE_SECRET_KEY"
export SCW_DEFAULT_PROJECT_ID="VOTRE_PROJECT_ID"
export SCW_DEFAULT_REGION="fr-par"
export SCW_DEFAULT_ZONE="fr-par-1"

export TF_VAR_scw_access_key="$SCW_ACCESS_KEY"
export TF_VAR_scw_secret_key="$SCW_SECRET_KEY"
export TF_VAR_scw_project_id="$SCW_DEFAULT_PROJECT_ID"

export PACKER_LOG=1
export PACKER_LOG_PATH="./packer.log"
```

Chargez les variables :

```bash
source .envrc
# Ou avec direnv
direnv allow
```

### 2. Création de l'image Talos

L'image Talos est créée via Packer en utilisant une instance temporaire :

```bash
cd packer

# Initialiser Packer
packer init .

# Valider la configuration
packer validate -var="talos_version=v1.11.5" .

# Build de l'image
packer build -var="talos_version=v1.11.5" .
```

L'image créée aura un nom comme : `talos-scaleway-v1.11.5-20251114-081824`

**Note** : Packer créera automatiquement :
1. Une instance temporaire
2. Téléchargera l'image Talos depuis Image Factory
3. Créera un snapshot
4. Générera une image réutilisable
5. Nettoiera les ressources temporaires

### 3. Déploiement de l'infrastructure

```bash
cd terraform

# Copier le fichier d'exemple de variables
cp terraform.tfvars.example terraform.tfvars

# Éditer avec vos paramètres (notamment l'image ID créée précédemment)
vim terraform.tfvars

# Initialiser Terraform
terraform init

# Planifier les changements
terraform plan

# Appliquer
terraform apply
```

### 4. Bootstrap du cluster Kubernetes

Une fois l'infrastructure déployée :

```bash
# Récupérer les outputs Terraform
export CONTROL_PLANE_IP=$(terraform output -raw control_plane_lb_ip)

# Générer les configurations Talos
talosctl gen config talos-cluster https://${CONTROL_PLANE_IP}:6443 \
  --output-dir _out \
  --with-docs=false \
  --with-examples=false

# Appliquer patch pour désactiver kube-proxy (Cilium le remplace)
talosctl --talosconfig _out/talosconfig machineconfig patch \
  _out/controlplane.yaml \
  --patch @cilium-patch.yaml \
  -o _out/controlplane-patched.yaml

# Appliquer les configurations aux control planes
for ip in $(terraform output -json control_plane_ips | jq -r '.[]'); do
  talosctl apply-config --insecure \
    --nodes $ip \
    --file _out/controlplane-patched.yaml
done

# Appliquer les configurations aux workers
for ip in $(terraform output -json worker_ips | jq -r '.[]'); do
  talosctl apply-config --insecure \
    --nodes $ip \
    --file _out/worker.yaml
done

# Bootstrap etcd sur le premier control plane
FIRST_CP=$(terraform output -json control_plane_ips | jq -r '.[0]')
talosctl --talosconfig _out/talosconfig \
  bootstrap --nodes $FIRST_CP

# Récupérer le kubeconfig
talosctl --talosconfig _out/talosconfig \
  kubeconfig _out/kubeconfig --nodes $CONTROL_PLANE_IP

# Vérifier le cluster
export KUBECONFIG=_out/kubeconfig
kubectl get nodes
```

### 5. Installation de Cilium

```bash
# Ajouter le repo Helm de Cilium
helm repo add cilium https://helm.cilium.io/
helm repo update

# Installer Cilium
helm install cilium cilium/cilium \
  --namespace kube-system \
  --set ipam.mode=kubernetes \
  --set kubeProxyReplacement=true \
  --set k8sServiceHost=$CONTROL_PLANE_IP \
  --set k8sServicePort=6443

# Vérifier l'installation
kubectl -n kube-system get pods -l k8s-app=cilium
cilium status --wait
```

🎉 **Votre cluster Kubernetes Talos sur Scaleway est opérationnel !**

## 📁 Structure du projet

```text
.
├── README.md                      # Ce fichier
├── .envrc.sample                  # Template de credentials
├── cilium-patch.yaml              # Patch Talos pour désactiver kube-proxy
├── packer/
│   ├── talos-scaleway.pkr.hcl    # Configuration Packer
│   ├── provision/
│   │   ├── build-image.sh         # Script de build de l'image
│   │   └── schematic.yaml         # Schematic Talos personnalisé
│   └── variables.pkr.hcl          # Variables Packer
├── terraform/
│   ├── main.tf                    # Configuration Terraform principale
│   ├── variables.tf               # Variables d'entrée
│   ├── outputs.tf                 # Outputs exposés
│   ├── versions.tf                # Provider versions
│   ├── vpc.tf                     # VPC et Private Networks
│   ├── compute.tf                 # Instances Talos
│   ├── security_groups.tf         # Security Groups
│   ├── load_balancer.tf           # Load Balancer API Kubernetes
│   ├── public_gateway.tf          # Public Gateway pour NAT
│   └── terraform.tfvars.example   # Exemple de variables
└── _out/                          # Outputs générés
    ├── talosconfig
    ├── kubeconfig
    ├── controlplane.yaml
    └── worker.yaml
```

## 📚 Documentation

### Documentation Scaleway

- **VPC & Private Networks** : https://www.scaleway.com/en/docs/vpc/
- **IPAM** : https://www.scaleway.com/en/docs/vpc/reference-content/understanding-ipam/
- **Instances** : https://www.scaleway.com/en/docs/compute/instances/
- **Load Balancers** : https://www.scaleway.com/en/docs/network/load-balancer/
- **Public Gateway** : https://www.scaleway.com/en/docs/public-gateways/

### Documentation Talos & Kubernetes

- **Talos officiel** : https://www.talos.dev/
- **Talos Image Factory** : https://factory.talos.dev/
- **Cilium** : https://docs.cilium.io/
- **Terraform Scaleway Provider** : https://registry.terraform.io/providers/scaleway/scaleway/

### Spécificités Scaleway

#### Private Networks et IPAM

- Les Private Networks sont **régionaux** et couvrent automatiquement toutes les AZ
- IPAM alloue automatiquement les IPs privées (pas de DHCP à configurer)
- CIDR par défaut : /22 (1020 IPs utilisables)
- DNS interne : `<resource-name>.<private-network-name>.internal`

#### Multi-AZ

- PAR-1, PAR-2, PAR-3 disponibles dans la région Paris
- Les instances sont zonales mais peuvent communiquer via le Private Network régional
- Les volumes Block Storage sont zonaux (non migrables entre AZ)

#### Public Gateway

- Fournit NAT pour accès internet sortant
- Mode IPAM obligatoire (legacy deprecated)
- Peut attacher jusqu'à 8 Private Networks
- SSH Bastion intégré disponible

## 🤝 Contribution

Les contributions sont les bienvenues !

1. Forker le projet
2. Créer une branche feature (`git checkout -b feature/AmazingFeature`)
3. Commiter vos changements (`git commit -m 'Add some AmazingFeature'`)
4. Pousser vers la branche (`git push origin feature/AmazingFeature`)
5. Ouvrir une Pull Request

## 📝 Licence

Ce projet est distribué sous licence Apache 2.0. Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## 👤 Auteur

**Adaptation Scaleway**

Basé sur le travail original pour Outscale par Stéphane Robert

---

⭐ **Si ce projet vous est utile, n'hésitez pas à lui mettre une star !**
