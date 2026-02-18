package xml

import (
	"encoding/xml"
	"fmt"
	"os"
)

type Domain struct {
	XMLName xml.Name `xml:"domain"`
	Type    string   `xml:"type,attr"`
	Name    string   `xml:"name"`
	Devices Devices  `xml:"devices"`
}

type Devices struct {
	Disks []Disk `xml:"disk"`
}

type Disk struct {
	Type   string `xml:"type,attr"`
	Device string `xml:"device,attr"`
	Driver Driver `xml:"driver"`
	Source Source `xml:"source"`
	Target Target `xml:"target"`
}

type Driver struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type Source struct {
	File string `xml:"file,attr"`
}

type Target struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr"`
}

func GenerateXML(diskFiles []string, outputFile, vmname string) error {
	var disks []Disk
	for i, file := range diskFiles {
		disk := Disk{
			Type:   "file",
			Device: "disk",
			Driver: Driver{Name: "qemu", Type: "raw"},
			Source: Source{File: file},
			Target: Target{Dev: fmt.Sprintf("sd%c", 'a'+i), Bus: "virtio"},
		}
		disks = append(disks, disk)
	}

	domain := Domain{
		Type:    "kvm",
		Name:    vmname,
		Devices: Devices{Disks: disks},
	}

	output, err := xml.MarshalIndent(domain, "", "  ")
	if err != nil {
		return err
	}

	outputFileHandle, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outputFileHandle.Close()

	_, err = outputFileHandle.Write([]byte(xml.Header + string(output)))
	return err
}
