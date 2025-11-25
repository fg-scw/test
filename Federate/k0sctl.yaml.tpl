apiVersion: k0sctl.k0sproject.io/v1beta1
kind: Cluster
metadata:
  name: k0s-multi-az-gpu
spec:
  hosts:
    # Controller 1 (fr-par-1) - aussi worker
    - role: controller
      ssh:
        address: ${controller_par1_ip}
        user: root
        keyPath: ${ssh_key_path}
      noTaints: true
      installFlags:
        - "--enable-worker"

    # Controller 2 (fr-par-2) - aussi worker
    - role: controller
      ssh:
        address: ${controller_par2_ip}
        user: root
        keyPath: ${ssh_key_path}
      noTaints: true
      installFlags:
        - "--enable-worker"

    # Worker GPU 1 (fr-par-1)
    - role: worker
      ssh:
        address: ${gpu_par1_ip}
        user: root
        keyPath: ${ssh_key_path}
      hooks:
        apply:
          before:
            - |
              bash -lc '
                set -euxo pipefail

                echo "[GPU-BOOTSTRAP] Installation et configuration driver NVIDIA + container toolkit"

                # Installation des packages si nécessaire
                if ! command -v nvidia-smi >/dev/null 2>&1; then
                  echo "[GPU-BOOTSTRAP] Installation driver NVIDIA..."
                  apt-get update
                  apt-get install -y ca-certificates curl gnupg

                  rm -f /etc/apt/sources.list.d/nvidia-container-toolkit.list || true
                  rm -f /etc/apt/sources.list.d/libnvidia-container.list || true

                  curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
                    | gpg --dearmor -o /etc/apt/trusted.gpg.d/nvidia-container-toolkit.gpg

                  curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
                    > /etc/apt/sources.list.d/nvidia-container-toolkit.list

                  apt-get update

                  DEBIAN_FRONTEND=noninteractive apt-get install -y \
                    nvidia-driver-535 \
                    nvidia-container-toolkit
                fi

                # Vérifier que nvidia-smi fonctionne
                if ! nvidia-smi >/dev/null 2>&1; then
                  echo "[GPU-BOOTSTRAP] ERREUR: nvidia-smi ne fonctionne pas"
                  exit 1
                fi
                
                echo "[GPU-BOOTSTRAP] Installation NVIDIA terminée avec succès"
              '
          after:
            - |
              bash -lc '
                set -euxo pipefail
                
                echo "[GPU-BOOTSTRAP] Post-install: Configuration du runtime NVIDIA dans k0s"
                
                # k0s utilise un système d import - on doit modifier le fichier importé
                CONTAINERD_CRI_TOML="/run/k0s/containerd-cri.toml"
                
                # Attendre que k0s crée le fichier
                for i in {1..30}; do
                  if [ -f "$$CONTAINERD_CRI_TOML" ]; then
                    break
                  fi
                  sleep 2
                done
                
                if [ ! -f "$$CONTAINERD_CRI_TOML" ]; then
                  echo "[GPU-BOOTSTRAP] ERREUR: $$CONTAINERD_CRI_TOML non créé par k0s"
                  exit 1
                fi
                
                # Vérifier si déjà configuré
                if grep -q "nvidia-container-runtime" "$$CONTAINERD_CRI_TOML"; then
                  echo "[GPU-BOOTSTRAP] Runtime NVIDIA déjà configuré"
                else
                  echo "[GPU-BOOTSTRAP] Ajout du runtime NVIDIA à $$CONTAINERD_CRI_TOML..."
                  
                  # Backup
                  cp "$$CONTAINERD_CRI_TOML" "$${CONTAINERD_CRI_TOML}.backup"
                  
                  # Ajouter le runtime NVIDIA
                  echo "" >> "$$CONTAINERD_CRI_TOML"
                  echo "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia]" >> "$$CONTAINERD_CRI_TOML"
                  echo "  runtime_type = \"io.containerd.runc.v2\"" >> "$$CONTAINERD_CRI_TOML"
                  echo "  [plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia.options]" >> "$$CONTAINERD_CRI_TOML"
                  echo "    BinaryName = \"/usr/bin/nvidia-container-runtime\"" >> "$$CONTAINERD_CRI_TOML"
                  
                  echo "[GPU-BOOTSTRAP] Runtime NVIDIA ajouté"
                fi
                
                # Redémarrer k0sworker pour charger la nouvelle config
                echo "[GPU-BOOTSTRAP] Redémarrage de k0sworker..."
                systemctl restart k0sworker
                sleep 5
                
                echo "[GPU-BOOTSTRAP] Configuration terminée, k0sworker redémarré"
              '


    # Worker GPU 2 (fr-par-2)
    - role: worker
      ssh:
        address: ${gpu_par2_ip}
        user: root
        keyPath: ${ssh_key_path}
      hooks:
        apply:
          before:
            - |
              bash -lc '
                set -euxo pipefail

                echo "[GPU-BOOTSTRAP] Installation et configuration driver NVIDIA + container toolkit"

                # Installation des packages si nécessaire
                if ! command -v nvidia-smi >/dev/null 2>&1; then
                  echo "[GPU-BOOTSTRAP] Installation driver NVIDIA..."
                  apt-get update
                  apt-get install -y ca-certificates curl gnupg

                  rm -f /etc/apt/sources.list.d/nvidia-container-toolkit.list || true
                  rm -f /etc/apt/sources.list.d/libnvidia-container.list || true

                  curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
                    | gpg --dearmor -o /etc/apt/trusted.gpg.d/nvidia-container-toolkit.gpg

                  curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
                    > /etc/apt/sources.list.d/nvidia-container-toolkit.list

                  apt-get update

                  DEBIAN_FRONTEND=noninteractive apt-get install -y \
                    nvidia-driver-535 \
                    nvidia-container-toolkit
                fi

                # Vérifier que nvidia-smi fonctionne
                if ! nvidia-smi >/dev/null 2>&1; then
                  echo "[GPU-BOOTSTRAP] ERREUR: nvidia-smi ne fonctionne pas"
                  exit 1
                fi
                
                echo "[GPU-BOOTSTRAP] Installation NVIDIA terminée avec succès"
              '
          after:
            - |
              bash -lc '
                set -euxo pipefail
                
                echo "[GPU-BOOTSTRAP] Post-install: Configuration du runtime NVIDIA dans k0s"
                
                # k0s utilise un système d import - on doit modifier le fichier importé
                CONTAINERD_CRI_TOML="/run/k0s/containerd-cri.toml"
                
                # Attendre que k0s crée le fichier
                for i in {1..30}; do
                  if [ -f "$$CONTAINERD_CRI_TOML" ]; then
                    break
                  fi
                  sleep 2
                done
                
                if [ ! -f "$$CONTAINERD_CRI_TOML" ]; then
                  echo "[GPU-BOOTSTRAP] ERREUR: $$CONTAINERD_CRI_TOML non créé par k0s"
                  exit 1
                fi
                
                # Vérifier si déjà configuré
                if grep -q "nvidia-container-runtime" "$$CONTAINERD_CRI_TOML"; then
                  echo "[GPU-BOOTSTRAP] Runtime NVIDIA déjà configuré"
                else
                  echo "[GPU-BOOTSTRAP] Ajout du runtime NVIDIA à $$CONTAINERD_CRI_TOML..."
                  
                  # Backup
                  cp "$$CONTAINERD_CRI_TOML" "$${CONTAINERD_CRI_TOML}.backup"
                  
                  # Ajouter le runtime NVIDIA
                  echo "" >> "$$CONTAINERD_CRI_TOML"
                  echo "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia]" >> "$$CONTAINERD_CRI_TOML"
                  echo "  runtime_type = \"io.containerd.runc.v2\"" >> "$$CONTAINERD_CRI_TOML"
                  echo "  [plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.nvidia.options]" >> "$$CONTAINERD_CRI_TOML"
                  echo "    BinaryName = \"/usr/bin/nvidia-container-runtime\"" >> "$$CONTAINERD_CRI_TOML"
                  
                  echo "[GPU-BOOTSTRAP] Runtime NVIDIA ajouté"
                fi
                
                # Redémarrer k0sworker pour charger la nouvelle config
                echo "[GPU-BOOTSTRAP] Redémarrage de k0sworker..."
                systemctl restart k0sworker
                sleep 5
                
                echo "[GPU-BOOTSTRAP] Configuration terminée, k0sworker redémarré"
              '


  k0s:
    version: v1.30.0+k0s.0
    config:
      apiVersion: k0s.k0sproject.io/v1beta1
      kind: ClusterConfig
      metadata:
        name: k0s-multi-az-gpu
      spec:
        network:
          provider: kuberouter
          nodeLocalLoadBalancing:
            enabled: true
            type: EnvoyProxy
