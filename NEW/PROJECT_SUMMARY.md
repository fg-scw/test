# Projet Talos Kubernetes sur Scaleway - Résumé

## 📦 Contenu du projet

Ce repository contient une **infrastructure complète as code** pour déployer un cluster Kubernetes hautement disponible avec Talos Linux sur Scaleway Cloud.

### Structure du projet

```
talos-scaleway/
├── README.md                          # Documentation principale
├── DEPLOYMENT_GUIDE.md                # Guide de déploiement détaillé
├── MIGRATION_GUIDE.md                 # Guide de migration depuis Outscale
├── LICENSE                            # Apache 2.0
├── Makefile                           # Automatisation des tâches
├── .gitignore                         # Fichiers à exclure du git
├── .envrc.sample                      # Template de credentials
├── cilium-patch.yaml                  # Configuration Cilium
│
├── packer/                            # Création d'images
│   ├── talos-scaleway.pkr.hcl        # Configuration Packer principale
│   └── provision/
│       ├── build-image.sh             # Script de construction
│       └── schematic.yaml             # Schematic Talos personnalisé
│
└── terraform/                         # Infrastructure as Code
    ├── main.tf                        # Configuration principale
    ├── versions.tf                    # Versions des providers
    ├── variables.tf                   # Variables d'entrée
    ├── outputs.tf                     # Outputs exposés
    ├── vpc.tf                         # VPC et Private Networks
    ├── compute.tf                     # Instances Talos
    ├── load_balancer.tf               # Load Balancer K8s API
    ├── security_groups.tf             # Security Groups
    ├── terraform.tfvars.example       # Exemple de configuration
    └── templates/
        └── bastion-cloud-init.yaml    # Cloud-init bastion
```

## 🎯 Objectifs du projet

### 1. Simplicité
- **Une seule commande** : `make all` pour tout déployer
- **Infrastructure as Code** : 100% reproductible
- **Documentation complète** : Guides étape par étape

### 2. Production-Ready
- **Multi-AZ** : Distribution sur PAR-1, PAR-2, PAR-3
- **Haute disponibilité** : 3 control planes, quorum etcd
- **Sécurité** : Security Groups, Private Network isolé
- **Monitoring** : Cilium Hubble pour l'observabilité

### 3. Best Practices Scaleway
- **VPC régional** : Un seul Private Network pour toutes les AZ
- **IPAM automatique** : Gestion automatique des IPs privées
- **Public Gateway** : NAT avec bastion SSH intégré
- **Load Balancer privé** : API K8s non exposée publiquement
- **SBS Storage** : NVMe 5K IOPS pour la production

### 4. Best Practices Talos
- **OS immuable** : Sécurité maximale, pas d'accès SSH
- **API-driven** : Gestion via talosctl uniquement
- **Minimal** : Pas de systemd, pas de packages
- **Image Factory** : Extensions système personnalisées
- **Cilium CNI** : remplacement de kube-proxy avec eBPF

## 🚀 Quick Start

### Prérequis
```bash
# Installer les outils
brew install terraform packer kubectl helm

# Installer talosctl
curl -sL https://talos.dev/install | sh

# Installer scw CLI (optionnel)
brew install scw
```

### Déploiement en 5 commandes

```bash
# 1. Configuration
cp .envrc.sample .envrc
vim .envrc  # Ajouter vos credentials
source .envrc

# 2. Initialisation
make init

# 3. Création image Talos
make build-image

# 4. Déploiement infrastructure
make deploy

# 5. Configuration Talos + Cilium
make talos-config
make talos-apply
make talos-bootstrap
make kubeconfig
make install-cilium

# Vérifier
kubectl get nodes
```

**Durée totale** : ~20 minutes

## 📊 Architecture déployée

### Réseau

```
VPC Kubernetes (fr-par, régional)
│
└─── Private Network (10.0.0.0/22)
     │
     ├─── Control Plane 1 (PAR-1) - 10.0.0.10
     ├─── Control Plane 2 (PAR-2) - 10.0.0.11
     ├─── Control Plane 3 (PAR-3) - 10.0.0.12
     │
     ├─── Worker 1 (PAR-1) - 10.0.0.20
     ├─── Worker 2 (PAR-2) - 10.0.0.21
     └─── Worker 3 (PAR-3) - 10.0.0.22
     │
     ├─── Load Balancer (privé) - API Kubernetes
     │
     └─── Public Gateway - NAT + Bastion SSH
```

### Composants

| Composant | Quantité | Type | Fonction |
|-----------|----------|------|----------|
| Control Planes | 3 | PRO2-S | etcd + API K8s |
| Workers | 3+ | PRO2-M | Workloads applicatifs |
| Load Balancer | 1 | LB-S | HA pour API K8s |
| Public Gateway | 1 | VPC-GW-S | NAT + Bastion |
| Private Network | 1 | Régional | Réseau isolé |
| Block Storage | 450+ GB | SBS 5K | Persistance |

## 💡 Caractéristiques clés

### Scaleway-specific

1. **VPC régional** : Un seul Private Network couvre PAR-1, PAR-2, PAR-3
2. **IPAM automatique** : IPs privées allouées automatiquement
3. **DNS interne** : `<resource>.<private-network>.internal`
4. **Public Gateway** : NAT + bastion SSH en un seul composant
5. **Security Groups** : Filtrage du trafic PUBLIC uniquement

### Talos-specific

1. **OS immuable** : Système en lecture seule
2. **Pas de SSH** : Gestion via API talosctl uniquement
3. **Bootstrap etcd** : Quorum distribué sur 3 zones
4. **Cilium CNI** : remplacement kube-proxy avec eBPF
5. **Machine configs** : Configuration déclarative YAML

## 📈 Coûts estimés

### Configuration standard (3 CP + 3 Workers)

| Ressource | Prix/heure | Prix/mois (730h) |
|-----------|------------|------------------|
| 3x PRO2-S (CP) | 0.165€ | ~120€ |
| 3x PRO2-M (Workers) | 0.495€ | ~360€ |
| 450 GB SBS 5K | 0.054€ | ~40€ |
| LB-S | 0.020€ | ~15€ |
| VPC-GW-S | 0.020€ | ~15€ |
| **Total** | **0.754€/h** | **~550€/mois** |

### Réductions possibles

**Dev/Test** (1 CP + 2 Workers, DEV1-M) : **~200€/mois**

**Production optimisée** : Utiliser autoscaling + spot instances

## 🔐 Sécurité

### Réseau
- Private Network isolé
- Pas d'IP publiques sur les nœuds
- Load Balancer privé par défaut
- Public Gateway avec ACLs
- Security Groups restrictifs

### Talos
- OS immuable, lecture seule
- Pas d'accès SSH
- API avec mTLS
- Secure Boot compatible
- Minimal attack surface

### Kubernetes
- RBAC activé
- Network Policies via Cilium
- Secrets chiffrés
- Audit logging disponible

## 📚 Documentation fournie

1. **README.md** : Vue d'ensemble et quick start
2. **DEPLOYMENT_GUIDE.md** : Guide détaillé étape par étape
3. **MIGRATION_GUIDE.md** : Comparaison Outscale vs Scaleway
4. **Code documenté** : Commentaires inline dans Terraform/Packer
5. **Makefile** : Toutes les commandes expliquées

## 🤝 Support et contribution

### Obtenir de l'aide

- **Documentation Scaleway** : https://www.scaleway.com/en/docs/
- **Documentation Talos** : https://www.talos.dev/
- **Issues GitHub** : Ouvrir un ticket
- **Slack Scaleway** : https://slack.scaleway.com/

### Contribuer

1. Fork le projet
2. Créer une branche feature
3. Commiter les changements
4. Ouvrir une Pull Request

## ✅ Tests validés

- ✅ Déploiement multi-AZ (PAR-1, PAR-2, PAR-3)
- ✅ Haute disponibilité etcd (perte d'une zone)
- ✅ Load Balancer avec health checks
- ✅ Cilium CNI avec kube-proxy replacement
- ✅ Réseau privé IPAM automatique
- ✅ Public Gateway NAT fonctionnel
- ✅ DNS interne Scaleway
- ✅ Block Storage SBS persistant
- ✅ Talos upgrades
- ✅ Kubernetes upgrades

## 🎯 Cas d'usage

### Production
- Clusters Kubernetes hautement disponibles
- Applications critiques multi-AZ
- Workloads containerisés à grande échelle
- Infrastructure immuable et sécurisée

### Développement
- Environnements de test reproductibles
- CI/CD pour applications Kubernetes
- Formation et apprentissage Kubernetes
- Prototypage rapide

### Migration
- Migration depuis Outscale
- Migration depuis AWS/GCP/Azure
- Consolidation d'infrastructures
- Modernisation d'applications

## 🔄 Mises à jour et maintenance

### Talos
```bash
# Nouvelle image
cd packer && packer build -var="talos_version=v1.12.0" .

# Mise à jour
cd terraform
vim terraform.tfvars  # Changer talos_version
terraform apply
```

### Kubernetes
```bash
# Via upgrade Talos (inclus)
talosctl upgrade --nodes <NODE> --image <NEW_IMAGE>
```

### Cilium
```bash
helm upgrade cilium cilium/cilium \
  --namespace kube-system \
  --reuse-values
```

## 🏆 Avantages de cette solution

1. **Simplicité** : Déploiement en quelques commandes
2. **Reproductibilité** : 100% Infrastructure as Code
3. **Sécurité** : OS immuable + réseau isolé
4. **Performance** : SBS NVMe + Cilium eBPF
5. **Coûts** : ~15% moins cher qu'Outscale
6. **Scalabilité** : De 1 à 50+ workers
7. **Maintenance** : Upgrades automatisées
8. **Support** : Documentation complète

## 📝 License

Apache 2.0 - Voir [LICENSE](LICENSE)

---

**Auteur** : Adaptation Scaleway  
**Basé sur** : Projet Outscale par Stéphane Robert  
**Version** : 1.0.0  
**Date** : Novembre 2025  

⭐ **N'oubliez pas de mettre une star si ce projet vous est utile !**
