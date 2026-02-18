// Copyright © 2024 The vjailbreak authors

package openstack

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
)

// GetClosestFlavour gets the closest flavor for the given CPU, memory, and GPU requirements.
// useGPUFlavor controls GPU flavor filtering:
//   - true: Only consider GPU-enabled flavors (even if GPU count = 0)
//   - false: Strictly omit GPU-enabled flavors unless GPU count > 0
// passthroughGPUCount and vgpuCount specify the required number of passthrough and vGPU devices.
func GetClosestFlavour(cpu, memory, passthroughGPUCount, vgpuCount int, allFlavors []flavors.Flavor, useGPUFlavor bool) (*flavors.Flavor, error) {
	// Check if the flavor slice is empty
	if len(allFlavors) == 0 {
		return nil, fmt.Errorf("no flavors available to select from")
	}

	bestFlavor := new(flavors.Flavor)
	bestFlavor.VCPUs = constants.MaxVCPUs
	bestFlavor.RAM = constants.MaxRAM

	// Find the smallest flavor that meets the requirements
	for _, flavor := range allFlavors {
		// Filter based on GPU flavor requirement
		if useGPUFlavor {
			// Only consider GPU-enabled flavors
			if !isGPUFlavor(flavor) {
				continue
			}
		} else {
			// Strictly omit GPU-enabled flavors unless GPU count is explicitly required
			if passthroughGPUCount == 0 && vgpuCount == 0 && isGPUFlavor(flavor) {
				continue
			}
		}

		// Check GPU requirements
		if passthroughGPUCount > 0 || vgpuCount > 0 {
			flavorPassthrough := getPassthroughGPUCount(flavor)
			flavorVGPU := getVGPUCount(flavor)

			// Flavor must meet GPU requirements
			if flavorPassthrough < passthroughGPUCount || flavorVGPU < vgpuCount {
				continue
			}
		}

		if flavor.VCPUs >= cpu && flavor.RAM >= memory {
			if flavor.VCPUs < bestFlavor.VCPUs ||
				(flavor.VCPUs == bestFlavor.VCPUs && flavor.RAM < bestFlavor.RAM) {
				bestFlavor = &flavor
			}
		}
	}

	if bestFlavor.VCPUs != constants.MaxVCPUs {
		return bestFlavor, nil
	}

	gpuInfo := ""
	if passthroughGPUCount > 0 || vgpuCount > 0 {
		gpuInfo = fmt.Sprintf(", %d passthrough GPU(s), and %d vGPU(s)", passthroughGPUCount, vgpuCount)
	}
	return nil, fmt.Errorf("no suitable flavor found for %d vCPU(s), %d MB RAM%s", cpu, memory, gpuInfo)
}

// isGPUFlavor checks if a flavor has GPU-related extra_specs
func isGPUFlavor(flavor flavors.Flavor) bool {
	if flavor.ExtraSpecs == nil {
		return false
	}

	for key := range flavor.ExtraSpecs {
		if key == "pci_passthrough:alias" ||
			strings.Contains(key, "trait:") ||
			key == "resources:VGPU" {
			return true
		}
	}
	return false
}

// getPassthroughGPUCount extracts the passthrough GPU count from flavor extra_specs.
// Example: "pci_passthrough:alias" = "nvidia-l4:1" returns 1
func getPassthroughGPUCount(flavor flavors.Flavor) int {
	if flavor.ExtraSpecs == nil {
		return 0
	}

	for key, value := range flavor.ExtraSpecs {
		if key == "pci_passthrough:alias" {
			// Format: "alias_name:count" e.g., "nvidia-l4:1"
			parts := strings.Split(value, ":")
			if len(parts) == 2 {
				if count, err := strconv.Atoi(parts[1]); err == nil {
					return count
				}
			}
		}
	}
	return 0
}

// getVGPUCount extracts the vGPU count from flavor extra_specs.
// Example: "resources:VGPU" = "1" returns 1
func getVGPUCount(flavor flavors.Flavor) int {
	if flavor.ExtraSpecs == nil {
		return 0
	}

	if value, exists := flavor.ExtraSpecs["resources:VGPU"]; exists {
		if count, err := strconv.Atoi(value); err == nil {
			return count
		}
	}
	return 0
}
