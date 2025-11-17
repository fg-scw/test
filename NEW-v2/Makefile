.PHONY: help packer-init packer-build terraform-init terraform-plan terraform-apply terraform-destroy clean

ZONE ?= fr-par-1
TALOS_VERSION ?= v1.11.5

help:
	@echo "Available targets:"
	@echo "  packer-init      - Initialize Packer plugins"
	@echo "  packer-build     - Build Talos image with Packer"
	@echo "  terraform-init   - Initialize Terraform"
	@echo "  terraform-plan   - Plan Terraform changes"
	@echo "  terraform-apply  - Apply Terraform configuration"
	@echo "  terraform-destroy - Destroy infrastructure"
	@echo "  clean            - Clean temporary files"
	@echo ""
	@echo "Variables:"
	@echo "  ZONE=$(ZONE)"
	@echo "  TALOS_VERSION=$(TALOS_VERSION)"

packer-init:
	cd packer && packer init .

packer-build: packer-init
	cd packer && packer build \
		-var "zone=$(ZONE)" \
		-var "talos_version=$(TALOS_VERSION)" \
		talos-scaleway.pkr.hcl

terraform-init:
	cd terraform && terraform init

terraform-plan: terraform-init
	cd terraform && terraform plan

terraform-apply: terraform-init
	cd terraform && terraform apply

terraform-destroy:
	cd terraform && terraform destroy

clean:
	rm -rf packer/packer_cache
	rm -f packer/manifest.json
	rm -f packer/packer.log
	rm -rf terraform/.terraform
	rm -f terraform/.terraform.lock.hcl
	rm -f terraform/terraform.tfstate*
