.PHONY: help init build-image deploy destroy clean validate test

# Variables
TALOS_VERSION ?= v1.11.5
CLUSTER_NAME ?= talos-prod

# Colors
CYAN := \033[0;36m
GREEN := \033[0;32m
RED := \033[0;31m
RESET := \033[0m

help: ## Afficher cette aide
	@echo "$(CYAN)Talos Kubernetes sur Scaleway$(RESET)"
	@echo ""
	@echo "$(GREEN)Commandes disponibles:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

init: ## Initialiser l'environnement (Packer + Terraform)
	@echo "$(GREEN)Initialisation de Packer...$(RESET)"
	cd packer && packer init .
	@echo "$(GREEN)Initialisation de Terraform...$(RESET)"
	cd terraform && terraform init
	@echo "$(GREEN)✓ Initialisation terminée$(RESET)"

build-image: ## Construire l'image Talos avec Packer
	@echo "$(GREEN)Construction de l'image Talos $(TALOS_VERSION)...$(RESET)"
	cd packer && packer build -var="talos_version=$(TALOS_VERSION)" .
	@echo "$(GREEN)✓ Image Talos créée avec succès$(RESET)"

validate: ## Valider les configurations Terraform et Packer
	@echo "$(GREEN)Validation Packer...$(RESET)"
	cd packer && packer validate -var="talos_version=$(TALOS_VERSION)" .
	@echo "$(GREEN)Validation Terraform...$(RESET)"
	cd terraform && terraform validate
	@echo "$(GREEN)✓ Validation réussie$(RESET)"

plan: ## Planifier les changements Terraform
	@echo "$(GREEN)Planification Terraform...$(RESET)"
	cd terraform && terraform plan

deploy: ## Déployer l'infrastructure complète
	@echo "$(GREEN)Déploiement de l'infrastructure...$(RESET)"
	cd terraform && terraform apply
	@echo "$(GREEN)✓ Infrastructure déployée$(RESET)"
	@echo ""
	@echo "$(CYAN)Prochaines étapes:$(RESET)"
	@echo "  1. make talos-config"
	@echo "  2. make talos-apply"
	@echo "  3. make talos-bootstrap"
	@echo "  4. make kubeconfig"
	@echo "  5. make install-cilium"

talos-config: ## Générer les configurations Talos
	@echo "$(GREEN)Génération des configurations Talos...$(RESET)"
	mkdir -p terraform/_out
	$(eval K8S_API := $(shell cd terraform && terraform output -raw kubernetes_api_endpoint))
	talosctl gen config $(CLUSTER_NAME) $(K8S_API) \
		--output-dir terraform/_out \
		--with-docs=false \
		--with-examples=false
	@echo "$(GREEN)Patch Cilium...$(RESET)"
	talosctl --talosconfig terraform/_out/talosconfig machineconfig patch \
		terraform/_out/controlplane.yaml \
		--patch @cilium-patch.yaml \
		-o terraform/_out/controlplane-patched.yaml
	@echo "$(GREEN)✓ Configurations générées dans terraform/_out/$(RESET)"

talos-apply: ## Appliquer les configurations Talos aux nœuds
	@echo "$(GREEN)Application des configurations Talos...$(RESET)"
	@echo "$(CYAN)Control Planes...$(RESET)"
	@cd terraform && \
	for ip in $$(terraform output -json control_plane_ips | jq -r '.[]'); do \
		echo "  → Applying to $$ip"; \
		talosctl apply-config --insecure --nodes $$ip --file _out/controlplane-patched.yaml; \
	done
	@echo "$(CYAN)Workers...$(RESET)"
	@cd terraform && \
	for ip in $$(terraform output -json worker_ips | jq -r '.[]'); do \
		echo "  → Applying to $$ip"; \
		talosctl apply-config --insecure --nodes $$ip --file _out/worker.yaml; \
	done
	@echo "$(GREEN)✓ Configurations appliquées$(RESET)"
	@echo "$(CYAN)Attendre 2-3 minutes que les nœuds démarrent...$(RESET)"

talos-bootstrap: ## Bootstrap etcd (une seule fois!)
	@echo "$(GREEN)Bootstrap etcd...$(RESET)"
	$(eval FIRST_CP := $(shell cd terraform && terraform output -json control_plane_ips | jq -r '.[0]'))
	@echo "  → Bootstrap sur $(FIRST_CP)"
	cd terraform && talosctl --talosconfig _out/talosconfig bootstrap --nodes $(FIRST_CP)
	@echo "$(GREEN)✓ etcd bootstrappé$(RESET)"

kubeconfig: ## Récupérer le kubeconfig
	@echo "$(GREEN)Récupération du kubeconfig...$(RESET)"
	$(eval K8S_API_IP := $(shell cd terraform && terraform output -raw kubernetes_api_ip))
	cd terraform && talosctl --talosconfig _out/talosconfig \
		kubeconfig _out/kubeconfig --nodes $(K8S_API_IP)
	@echo "$(GREEN)✓ Kubeconfig sauvegardé dans terraform/_out/kubeconfig$(RESET)"
	@echo ""
	@echo "$(CYAN)Pour utiliser kubectl:$(RESET)"
	@echo "  export KUBECONFIG=$$(pwd)/terraform/_out/kubeconfig"
	@echo "  kubectl get nodes"

install-cilium: ## Installer Cilium CNI
	@echo "$(GREEN)Installation de Cilium...$(RESET)"
	$(eval K8S_API_IP := $(shell cd terraform && terraform output -raw kubernetes_api_ip))
	helm repo add cilium https://helm.cilium.io/ || true
	helm repo update
	helm install cilium cilium/cilium \
		--namespace kube-system \
		--set ipam.mode=kubernetes \
		--set kubeProxyReplacement=true \
		--set k8sServiceHost=$(K8S_API_IP) \
		--set k8sServicePort=6443 \
		--set hubble.relay.enabled=true \
		--set hubble.ui.enabled=true
	@echo "$(GREEN)✓ Cilium installé$(RESET)"
	@echo "$(CYAN)Vérifier l'installation:$(RESET)"
	@echo "  kubectl -n kube-system get pods -l k8s-app=cilium"
	@echo "  cilium status --wait"

test: ## Tester le cluster
	@echo "$(GREEN)Tests du cluster...$(RESET)"
	@echo "$(CYAN)Nœuds:$(RESET)"
	kubectl get nodes
	@echo ""
	@echo "$(CYAN)Pods système:$(RESET)"
	kubectl -n kube-system get pods
	@echo ""
	@echo "$(CYAN)Status Cilium:$(RESET)"
	cilium status || echo "$(RED)cilium CLI non installé$(RESET)"

status: ## Afficher le status du cluster
	@echo "$(CYAN)═══════════════════════════════════════════$(RESET)"
	@echo "$(GREEN)Status du cluster $(CLUSTER_NAME)$(RESET)"
	@echo "$(CYAN)═══════════════════════════════════════════$(RESET)"
	@cd terraform && terraform output next_steps || echo "$(RED)Infrastructure non déployée$(RESET)"

destroy: ## Détruire l'infrastructure (ATTENTION: suppression définitive!)
	@echo "$(RED)⚠️  ATTENTION: Cela va SUPPRIMER DÉFINITIVEMENT le cluster!$(RESET)"
	@echo -n "Êtes-vous sûr? [y/N] " && read ans && [ $${ans:-N} = y ]
	@echo "$(RED)Destruction de l'infrastructure...$(RESET)"
	cd terraform && terraform destroy
	@echo "$(GREEN)✓ Infrastructure détruite$(RESET)"

clean: ## Nettoyer les fichiers générés
	@echo "$(GREEN)Nettoyage...$(RESET)"
	rm -rf terraform/.terraform/
	rm -rf terraform/_out/
	rm -f terraform/*.tfstate*
	rm -f packer/manifest.json
	rm -f packer/packer.log
	@echo "$(GREEN)✓ Nettoyage terminé$(RESET)"

outputs: ## Afficher les outputs Terraform
	@cd terraform && terraform output

logs: ## Afficher les logs d'un nœud
	@echo "$(CYAN)Logs Talos (premier control plane):$(RESET)"
	$(eval FIRST_CP := $(shell cd terraform && terraform output -json control_plane_ips | jq -r '.[0]'))
	cd terraform && talosctl --talosconfig _out/talosconfig -n $(FIRST_CP) logs

# Workflow complet
all: init validate build-image deploy talos-config talos-apply talos-bootstrap kubeconfig install-cilium test ## Workflow complet de A à Z

.DEFAULT_GOAL := help
