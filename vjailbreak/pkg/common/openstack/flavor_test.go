// Copyright © 2024 The vjailbreak authors

package openstack

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
)

func TestGetPassthroughGPUCount(t *testing.T) {
	tests := []struct {
		name     string
		flavor   flavors.Flavor
		expected int
	}{
		{
			name: "passthrough GPU with count 1",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"pci_passthrough:alias": "nvidia-l4:1",
				},
			},
			expected: 1,
		},
		{
			name: "passthrough GPU with count 2",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"pci_passthrough:alias": "nvidia-a100:2",
				},
			},
			expected: 2,
		},
		{
			name: "no passthrough GPU",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"resources:VGPU": "1",
				},
			},
			expected: 0,
		},
		{
			name: "invalid format",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"pci_passthrough:alias": "nvidia-l4",
				},
			},
			expected: 0,
		},
		{
			name:     "nil extra_specs",
			flavor:   flavors.Flavor{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPassthroughGPUCount(tt.flavor)
			if result != tt.expected {
				t.Errorf("getPassthroughGPUCount() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetVGPUCount(t *testing.T) {
	tests := []struct {
		name     string
		flavor   flavors.Flavor
		expected int
	}{
		{
			name: "vGPU with count 1",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"resources:VGPU": "1",
				},
			},
			expected: 1,
		},
		{
			name: "vGPU with count 2",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"resources:VGPU": "2",
				},
			},
			expected: 2,
		},
		{
			name: "no vGPU",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"pci_passthrough:alias": "nvidia-l4:1",
				},
			},
			expected: 0,
		},
		{
			name: "invalid format",
			flavor: flavors.Flavor{
				ExtraSpecs: map[string]string{
					"resources:VGPU": "invalid",
				},
			},
			expected: 0,
		},
		{
			name:     "nil extra_specs",
			flavor:   flavors.Flavor{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getVGPUCount(tt.flavor)
			if result != tt.expected {
				t.Errorf("getVGPUCount() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetClosestFlavourWithGPU(t *testing.T) {
	allFlavors := []flavors.Flavor{
		{
			ID:    "flavor-1",
			Name:  "m1.small",
			VCPUs: 2,
			RAM:   2048,
		},
		{
			ID:    "flavor-2",
			Name:  "gpu.small",
			VCPUs: 4,
			RAM:   4096,
			ExtraSpecs: map[string]string{
				"pci_passthrough:alias": "nvidia-l4:1",
			},
		},
		{
			ID:    "flavor-3",
			Name:  "vgpu.small",
			VCPUs: 4,
			RAM:   4096,
			ExtraSpecs: map[string]string{
				"resources:VGPU": "1",
			},
		},
		{
			ID:    "flavor-4",
			Name:  "gpu.large",
			VCPUs: 8,
			RAM:   8192,
			ExtraSpecs: map[string]string{
				"pci_passthrough:alias": "nvidia-a100:2",
			},
		},
	}

	tests := []struct {
		name                string
		cpu                 int
		memory              int
		passthroughGPUCount int
		vgpuCount           int
		useGPUFlavor        bool
		expectedFlavorID    string
		expectError         bool
	}{
		{
			name:                "match passthrough GPU count 1",
			cpu:                 2,
			memory:              2048,
			passthroughGPUCount: 1,
			vgpuCount:           0,
			useGPUFlavor:        false,
			expectedFlavorID:    "flavor-2",
			expectError:         false,
		},
		{
			name:                "match vGPU count 1",
			cpu:                 2,
			memory:              2048,
			passthroughGPUCount: 0,
			vgpuCount:           1,
			useGPUFlavor:        false,
			expectedFlavorID:    "flavor-3",
			expectError:         false,
		},
		{
			name:                "match passthrough GPU count 2",
			cpu:                 4,
			memory:              4096,
			passthroughGPUCount: 2,
			vgpuCount:           0,
			useGPUFlavor:        false,
			expectedFlavorID:    "flavor-4",
			expectError:         false,
		},
		{
			name:                "no GPU required - should omit GPU flavors",
			cpu:                 2,
			memory:              2048,
			passthroughGPUCount: 0,
			vgpuCount:           0,
			useGPUFlavor:        false,
			expectedFlavorID:    "flavor-1",
			expectError:         false,
		},
		{
			name:                "no GPU required but useGPUFlavor=true - should select GPU flavor",
			cpu:                 2,
			memory:              2048,
			passthroughGPUCount: 0,
			vgpuCount:           0,
			useGPUFlavor:        true,
			expectedFlavorID:    "flavor-2",
			expectError:         false,
		},
		{
			name:                "GPU requirement not met",
			cpu:                 2,
			memory:              2048,
			passthroughGPUCount: 3,
			vgpuCount:           0,
			useGPUFlavor:        false,
			expectedFlavorID:    "",
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flavor, err := GetClosestFlavour(tt.cpu, tt.memory, tt.passthroughGPUCount, tt.vgpuCount, allFlavors, tt.useGPUFlavor)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if flavor.ID != tt.expectedFlavorID {
				t.Errorf("GetClosestFlavour() = %s, want %s", flavor.ID, tt.expectedFlavorID)
			}
		})
	}
}
