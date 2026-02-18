// Copyright © 2024 The vjailbreak authors

package vcenter

import (
	"context"
	"log"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25/soap"
)

func simulateVCenter() (*VCenterClient, *simulator.Model, *simulator.Server, error) {
	// Create a new simulator instance
	model := simulator.VPX()
	err := model.Create()
	if err != nil {
		log.Fatal(err)
	}
	server := model.Service.NewServer()

	// Connect to the simulator
	u, err := soap.ParseURL(server.URL.String())
	if err != nil {
		log.Fatal(err)
	}
	u.User = url.UserPassword("user", "pass")
	ctx := context.Background()

	// Create a new client
	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		log.Fatal(err)
	}
	return &VCenterClient{
		VCClient:            client.Client,
		VCFinder:            find.NewFinder(client.Client, false),
		VCPropertyCollector: property.DefaultCollector(client.Client),
	}, model, server, nil
}

func cleanupSimulator(model *simulator.Model, server *simulator.Server) {
	model.Remove()
	server.Close()
}

func TestGetVMByName(t *testing.T) {
	simVC, model, server, err := simulateVCenter()
	defer cleanupSimulator(model, server)
	assert.Nil(t, err)

	// Test for a VM that doesn't exist
	vmName := "i_dont_exist"
	vm, err := simVC.GetVMByName(context.TODO(), vmName)
	assert.Nil(t, vm)
	assert.EqualError(t, err, "VM not found")

	// Test for a VM that does exist
	vmName = "DC0_H0_VM0"
	vm, err = simVC.GetVMByName(context.TODO(), vmName)
	assert.Nil(t, err)
	assert.IsType(t, &object.VirtualMachine{}, vm)
	assert.Equal(t, vmName, vm.Name())
}

func TestGetThumbprint(t *testing.T) {
	url := "www.google.com"
	thumbprint, err := GetThumbprint(url)
	assert.Nil(t, err)
	assert.NotEmpty(t, thumbprint)

	url = "poimyusfdv.com"
	thumbprint, err = GetThumbprint(url)
	assert.ErrorContains(t, err, "failed to connect to vCenter")
	assert.Empty(t, thumbprint)
}
