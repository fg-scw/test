package xml

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateXML(t *testing.T) {
	tests := []struct {
		name       string
		diskFiles  []string
		outputFile string
		vmname     string
		expected   Domain
	}{
		{
			name:       "single disk",
			diskFiles:  []string{"disk1.img"},
			outputFile: "test_output_1.xml",
			vmname:     "test-vm-1",
			expected: Domain{
				XMLName: xml.Name{Local: "domain"},
				Type:    "kvm",
				Name:    "test-vm-1",
				Devices: Devices{
					Disks: []Disk{
						{
							Type:   "file",
							Device: "disk",
							Driver: Driver{Name: "qemu", Type: "raw"},
							Source: Source{File: "disk1.img"},
							Target: Target{Dev: "sda", Bus: "virtio"},
						},
					},
				},
			},
		},
		{
			name:       "multiple disks",
			diskFiles:  []string{"disk1.img", "disk2.img"},
			outputFile: "test_output_2.xml",
			vmname:     "test-vm-2",
			expected: Domain{
				XMLName: xml.Name{Local: "domain"},
				Type:    "kvm",
				Name:    "test-vm-2",
				Devices: Devices{
					Disks: []Disk{
						{
							Type:   "file",
							Device: "disk",
							Driver: Driver{Name: "qemu", Type: "raw"},
							Source: Source{File: "disk1.img"},
							Target: Target{Dev: "sda", Bus: "virtio"},
						},
						{
							Type:   "file",
							Device: "disk",
							Driver: Driver{Name: "qemu", Type: "raw"},
							Source: Source{File: "disk2.img"},
							Target: Target{Dev: "sdb", Bus: "virtio"},
						},
					},
				},
			},
		},
		{
			name:       "no disks",
			diskFiles:  []string{},
			outputFile: "test_output_3.xml",
			vmname:     "test-vm-3",
			expected: Domain{
				XMLName: xml.Name{Local: "domain"},
				Type:    "kvm",
				Name:    "test-vm-3",
				Devices: Devices{Disks: nil},
			},
		},
		{
			name:       "three disks",
			diskFiles:  []string{"disk1.img", "disk2.img", "disk3.img"},
			outputFile: "test_output_4.xml",
			vmname:     "test-vm-4",
			expected: Domain{
				XMLName: xml.Name{Local: "domain"},
				Type:    "kvm",
				Name:    "test-vm-4",
				Devices: Devices{
					Disks: []Disk{
						{
							Type:   "file",
							Device: "disk",
							Driver: Driver{Name: "qemu", Type: "raw"},
							Source: Source{File: "disk1.img"},
							Target: Target{Dev: "sda", Bus: "virtio"},
						},
						{
							Type:   "file",
							Device: "disk",
							Driver: Driver{Name: "qemu", Type: "raw"},
							Source: Source{File: "disk2.img"},
							Target: Target{Dev: "sdb", Bus: "virtio"},
						},
						{
							Type:   "file",
							Device: "disk",
							Driver: Driver{Name: "qemu", Type: "raw"},
							Source: Source{File: "disk3.img"},
							Target: Target{Dev: "sdc", Bus: "virtio"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateXML(tt.diskFiles, tt.outputFile, tt.vmname)
			assert.NoError(t, err)

			output, err := ioutil.ReadFile(tt.outputFile)
			assert.NoError(t, err)

			var domain Domain
			err = xml.Unmarshal(output, &domain)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, domain)

			os.Remove(tt.outputFile)
		})
	}
}
