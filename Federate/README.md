# k0s + KubeRay + vLLM (multi-AZ Scaleway, L4-1-24G)

Cette v2 déploie :
- 3 noeuds contrôleurs k0s (HA) : `fr-par-1`, `fr-par-2`, `pl-waw-2`
- 3 noeuds workers GPU `L4-1-24G` (un par AZ)
- Un cluster k0s multi-AZ
- L'operator KubeRay
- Un `RayService` vLLM OpenAI-compatible réparti sur les 3 AZ

## Fichiers principaux

- `main.tf`, `variables.tf`, `terraform.tfvars.example.tfvars` : infrastructure Scaleway (instances + IP + SG)
- `k0sctl.yaml.tpl` : template k0sctl généré par Terraform
- `deploy.sh` : pipeline complet (Terraform -> k0sctl -> NVIDIA plugin -> KubeRay -> vLLM)
- `vllm-service.yaml` : ressource RayService (cluster Ray + vLLM)
- `test-vllm.sh` : script de test OpenAI-compatible
- `Dockerfile.ray-vllm` : exemple d'image Ray + vLLM custom
- `Makefile` : raccourcis pour les commandes principales

## Utilisation rapide

1. Copier `terraform.tfvars.example.tfvars` vers `terraform.tfvars` et renseigner :
   - `ssh_public_key`
   - `ssh_private_key_path` (clé ayant accès root aux instances)

2. Déployer :
   ```bash
   make init
   make deploy
   ```

3. Une fois le RayService en place, exposer l'API vLLM (port-forward, ingress, etc.) et tester :
   ```bash
   export VLLM_BASE_URL="http://localhost:8000/v1"
   make test
   ```

4. Pour tout détruire :
   ```bash
   make destroy
   ```

**Note sécurité :** les security groups sont volontairement ouverts (`accept`) pour simplifier le lab. Pour une prod réelle, il est fortement recommandé de les restreindre (IP source, ports nécessaires uniquement, Private Network, etc.).
