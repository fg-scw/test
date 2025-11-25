SHELL := /bin/bash
.PHONY: help init deploy status test destroy clean fix-labels fix-nvidia check-gpu

help:
	@echo "Makefile pour déploiement k0s + KubeRay + vLLM (multi-AZ Scaleway)"
	@echo ""
	@echo "Commandes principales:"
	@echo "  make init     - Initialiser Terraform"
	@echo "  make deploy   - Déploiement complet (infra + k0s + KubeRay + vLLM)"
	@echo "  make status   - Afficher l'état du cluster et des pods Ray/vLLM"
	@echo "  make test     - Tester l'API vLLM (nécessite port-forward actif)"
	@echo "  make destroy  - Détruire l'infra et réinitialiser k0s"
	@echo "  make clean    - Nettoyer les fichiers générés"
	@echo ""
	@echo "Commandes de dépannage:"
	@echo "  make fix-labels  - Ajouter les labels topology manquants"
	@echo "  make fix-nvidia  - Reconfigurer le runtime NVIDIA et redémarrer pods"
	@echo "  make check-gpu   - Vérifier l'exposition des GPUs"

init:
	@echo "Initialisation Terraform..."
	terraform init

deploy:
	@echo "Déploiement complet..."
	chmod +x deploy.sh
	./deploy.sh

status:
	@echo "État du cluster..."
	@if [ -f kubeconfig ]; then \
		export KUBECONFIG=$$(pwd)/kubeconfig; \
		echo "=== Nœuds ==="; \
		kubectl get nodes -o wide -L topology.kubernetes.io/zone,nvidia.com/gpu.present; \
		echo ""; \
		echo "=== GPUs exposés ==="; \
		kubectl get nodes -o custom-columns=NAME:.metadata.name,ZONE:.metadata.labels.'topology\.kubernetes\.io/zone',GPU-CAPACITY:.status.capacity.'nvidia\.com/gpu',GPU-ALLOCATABLE:.status.allocatable.'nvidia\.com/gpu'; \
		echo ""; \
		echo "=== Pods NVIDIA ==="; \
		kubectl get pods -n kube-system -l app.kubernetes.io/name=nvidia-device-plugin -o wide; \
		echo ""; \
		echo "=== Pods Ray/vLLM ==="; \
		kubectl get pods -A | grep -E "ray|vllm" || echo "Aucun pod Ray/vLLM trouvé"; \
		echo ""; \
		echo "=== RayServices ==="; \
		kubectl get rayservices.ray.io -A || echo "Aucun RayService trouvé"; \
	else \
		echo "kubeconfig introuvable. Lance d'abord 'make deploy'."; \
	fi

fix-labels:
	@echo "Ajout des labels topology.kubernetes.io/zone..."
	@if [ -f kubeconfig ]; then \
		export KUBECONFIG=$$(pwd)/kubeconfig; \
		kubectl label node k0s-controller-fr-par-1 topology.kubernetes.io/zone=fr-par-1 topology.kubernetes.io/region=fr-par --overwrite; \
		kubectl label node k0s-controller-fr-par-2 topology.kubernetes.io/zone=fr-par-2 topology.kubernetes.io/region=fr-par --overwrite; \
		kubectl label node k0s-gpu-fr-par-1 topology.kubernetes.io/zone=fr-par-1 topology.kubernetes.io/region=fr-par --overwrite; \
		kubectl label node k0s-gpu-fr-par-2 topology.kubernetes.io/zone=fr-par-2 topology.kubernetes.io/region=fr-par --overwrite; \
		echo "Labels ajoutés avec succès"; \
		kubectl get nodes -L topology.kubernetes.io/zone; \
	else \
		echo "kubeconfig introuvable"; \
	fi

fix-nvidia:
	@echo "Reconfiguration du runtime NVIDIA pour k0s..."
	@if [ ! -f terraform.tfvars ]; then echo "terraform.tfvars non trouvé"; exit 1; fi
	@GPU_IP1=$$(terraform output -raw gpu_par1_ip 2>/dev/null || echo ""); \
	GPU_IP2=$$(terraform output -raw gpu_par2_ip 2>/dev/null || echo ""); \
	if [ -n "$$GPU_IP1" ]; then \
		echo "Configuration de k0s-gpu-fr-par-1 ($$GPU_IP1)..."; \
		ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@$$GPU_IP1 '\
			set -e; \
			CONTAINERD_CRI_TOML="/run/k0s/containerd-cri.toml"; \
			if [ ! -f "$$CONTAINERD_CRI_TOML" ]; then echo "ERREUR: $$CONTAINERD_CRI_TOML inexistant"; exit 1; fi; \
			if grep -q "nvidia-container-runtime" "$$CONTAINERD_CRI_TOML"; then \
				echo "Runtime NVIDIA déjà configuré"; \
			else \
				echo "Ajout du runtime NVIDIA..."; \
				cp "$$CONTAINERD_CRI_TOML" "$${CONTAINERD_CRI_TOML}.backup"; \
				echo "" >> "$$CONTAINERD_CRI_TOML"; \
				echo "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia]" >> "$$CONTAINERD_CRI_TOML"; \
				echo "  runtime_type = \"io.containerd.runc.v2\"" >> "$$CONTAINERD_CRI_TOML"; \
				echo "  [plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia.options]" >> "$$CONTAINERD_CRI_TOML"; \
				echo "    BinaryName = \"/usr/bin/nvidia-container-runtime\"" >> "$$CONTAINERD_CRI_TOML"; \
				echo "Configuration ajoutée"; \
			fi; \
			systemctl restart k0sworker; \
			echo "k0sworker redémarré"; \
		' || echo "Erreur sur GPU node 1"; \
	fi; \
	if [ -n "$$GPU_IP2" ]; then \
		echo "Configuration de k0s-gpu-fr-par-2 ($$GPU_IP2)..."; \
		ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@$$GPU_IP2 '\
			set -e; \
			CONTAINERD_CRI_TOML="/run/k0s/containerd-cri.toml"; \
			if [ ! -f "$$CONTAINERD_CRI_TOML" ]; then echo "ERREUR: $$CONTAINERD_CRI_TOML inexistant"; exit 1; fi; \
			if grep -q "nvidia-container-runtime" "$$CONTAINERD_CRI_TOML"; then \
				echo "Runtime NVIDIA déjà configuré"; \
			else \
				echo "Ajout du runtime NVIDIA..."; \
				cp "$$CONTAINERD_CRI_TOML" "$${CONTAINERD_CRI_TOML}.backup"; \
				echo "" >> "$$CONTAINERD_CRI_TOML"; \
				echo "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia]" >> "$$CONTAINERD_CRI_TOML"; \
				echo "  runtime_type = \"io.containerd.runc.v2\"" >> "$$CONTAINERD_CRI_TOML"; \
				echo "  [plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia.options]" >> "$$CONTAINERD_CRI_TOML"; \
				echo "    BinaryName = \"/usr/bin/nvidia-container-runtime\"" >> "$$CONTAINERD_CRI_TOML"; \
				echo "Configuration ajoutée"; \
			fi; \
			systemctl restart k0sworker; \
			echo "k0sworker redémarré"; \
		' || echo "Erreur sur GPU node 2"; \
	fi
	@if [ -f kubeconfig ]; then \
		export KUBECONFIG=$$(pwd)/kubeconfig; \
		echo "Redémarrage des DaemonSets NVIDIA..."; \
		kubectl rollout restart daemonset/nvidia-device-plugin -n kube-system 2>/dev/null || true; \
		kubectl rollout restart daemonset/nvidia-device-plugin-gpu-feature-discovery -n kube-system 2>/dev/null || true; \
		echo "Attente de la stabilisation (60s)..."; \
		sleep 60; \
		echo "Recréation des pods Ray bloqués..."; \
		kubectl delete pods -n default -l ray.io/cluster --force --grace-period=0 2>/dev/null || true; \
		echo "Fix appliqué. Vérifiez avec: make check-gpu"; \
	fi

check-gpu:
	@echo "Vérification de l'exposition des GPUs..."
	@if [ -f kubeconfig ]; then \
		export KUBECONFIG=$$(pwd)/kubeconfig; \
		kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU-CAPACITY:.status.capacity.'nvidia\.com/gpu',GPU-ALLOCATABLE:.status.allocatable.'nvidia\.com/gpu'; \
	else \
		echo "kubeconfig introuvable"; \
	fi

test:
	@echo "Test de l'API vLLM (ex: http://localhost:8000/v1)..."
	@if [ -z "$$VLLM_BASE_URL" ]; then \
		echo "ERREUR: Variable VLLM_BASE_URL non définie"; \
		echo "Usage: VLLM_BASE_URL=http://localhost:8000/v1 make test"; \
		exit 1; \
	fi
	@./test-vllm.sh $$VLLM_BASE_URL

destroy:
	@echo "Destruction de l'infrastructure et reset k0s..."
	@read -p "Êtes-vous sûr? [y/N] " -n 1 -r; echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		if [ -f kubeconfig ]; then \
			export KUBECONFIG=$$(pwd)/kubeconfig; \
			kubectl delete -f vllm-service.yaml --ignore-not-found=true || true; \
		fi; \
		if [ -f k0sctl.yaml ]; then \
			./k0sctl reset --config k0sctl.yaml || true; \
		fi; \
		terraform destroy -auto-approve; \
	fi

clean:
	@echo "Nettoyage des fichiers générés..."
	rm -f k0sctl.yaml kubeconfig tfplan
	rm -rf .terraform .terraform.lock.hcl
