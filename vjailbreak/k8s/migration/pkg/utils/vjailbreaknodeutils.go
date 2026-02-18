package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/scope"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckAndCreateMasterNodeEntry ensures a master node entry exists and creates it if needed
func CheckAndCreateMasterNodeEntry(ctx context.Context, k3sclient client.Client, local bool) error {
	var openstackuuid string

	masterNode, err := GetMasterK8sNode(ctx, k3sclient)
	if err != nil {
		return errors.Wrap(err, "failed to get master node")
	}

	err = k3sclient.Get(ctx, client.ObjectKey{Name: masterNode.Name}, &vjailbreakv1alpha1.VjailbreakNode{})
	if err == nil {
		// VjailbreakNode already exists
		return nil
	}

	if local {
		// Local mode
		openstackuuid = "fake-openstackuuid"
	} else {
		// Controller manager is always on the master node due to pod affinity
		openstackuuid, err = utils.GetCurrentInstanceUUID()
		if err != nil {
			return errors.Wrap(err, "failed to get current instance uuid")
		}
	}
	vjNode := vjailbreakv1alpha1.VjailbreakNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VjailbreakMasterNodeName,
			Namespace: constants.NamespaceMigrationSystem,
		},
		Spec: vjailbreakv1alpha1.VjailbreakNodeSpec{
			NodeRole: constants.NodeRoleMaster,
		},
	}

	err = k3sclient.Create(ctx, &vjNode)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create vjailbreak node")
	}

	err = k3sclient.Get(ctx, types.NamespacedName{
		Namespace: constants.NamespaceMigrationSystem,
		Name:      constants.VjailbreakMasterNodeName,
	}, &vjNode)
	if err != nil {
		return errors.Wrap(err, "failed to get vjailbreak node")
	}

	vjNode.Status.VMIP = GetNodeInternalIP(masterNode)
	vjNode.Status.Phase = constants.VjailbreakNodePhaseNodeReady
	vjNode.Status.OpenstackUUID = openstackuuid

	err = k3sclient.Status().Update(ctx, &vjNode)
	if err != nil {
		return errors.Wrap(err, "failed to update vjailbreak node status")
	}

	return nil
}

// UpdateMasterNodeImageID updates the image ID of the master node
func UpdateMasterNodeImageID(ctx context.Context, k3sclient client.Client, local bool) error {
	var imageID string

	vjNode := vjailbreakv1alpha1.VjailbreakNode{}
	err := k3sclient.Get(ctx, types.NamespacedName{
		Namespace: constants.NamespaceMigrationSystem,
		Name:      constants.VjailbreakMasterNodeName,
	}, &vjNode)
	if err != nil {
		return errors.Wrap(err, "failed to get vjailbreak node")
	}

	openstackcreds, err := GetOpenstackCredsVjailbreakNode(ctx, k3sclient, &vjNode)
	if err != nil {
		return errors.Wrap(err, "failed to get openstack credentials for master")
	}

	if local {
		// Local mode
		imageID = "fake-image-id"
	} else {
		// Controller manager is always on the master node due to pod affinity
		imageID, err = GetImageIDFromVM(ctx, k3sclient, vjNode.Status.OpenstackUUID, openstackcreds)
		if err != nil {
			return errors.Wrap(err, "failed to get image id of master node")
		}
	}

	vjNode.Spec.OpenstackImageID = imageID
	vjNode.Spec.OpenstackCreds = corev1.ObjectReference{
		Name:      openstackcreds.Name,
		Namespace: openstackcreds.Namespace,
		Kind:      openstackcreds.Kind,
	}

	err = k3sclient.Update(ctx, &vjNode)
	if err != nil {
		return errors.Wrap(err, "failed to update vjailbreak node")
	}
	return nil
}

// IsMasterNode checks if the given node is a master node
func IsMasterNode(node *corev1.Node) bool {
	_, ok := node.Labels[constants.K8sMasterNodeAnnotation]
	return ok
}

// GetAllk8sNodes retrieves all Kubernetes nodes in the cluster
func GetAllk8sNodes(ctx context.Context, k3sclient client.Client) (corev1.NodeList, error) {
	nodeList := corev1.NodeList{}
	err := k3sclient.List(ctx, &nodeList)
	if err != nil {
		return corev1.NodeList{}, err
	}
	return nodeList, nil
}

// GetNodeInternalIP retrieves the internal IP address of a node
func GetNodeInternalIP(node *corev1.Node) string {
	return node.Annotations[constants.InternalIPAnnotation]
}

// GetMasterK8sNode retrieves the Kubernetes master node
func GetMasterK8sNode(ctx context.Context, k3sclient client.Client) (*corev1.Node, error) {
	nodeList, err := GetAllk8sNodes(ctx, k3sclient)
	if err != nil {
		return nil, err
	}

	var masterNode *corev1.Node

	for i := range nodeList.Items {
		if IsMasterNode(&nodeList.Items[i]) {
			masterNode = &nodeList.Items[i]
			break
		}
	}
	if masterNode == nil {
		return nil, fmt.Errorf("node with required annotation not found")
	}
	return masterNode, nil
}

// CreateOpenstackVMForWorkerNode creates a new OpenStack VM for a worker node
func CreateOpenstackVMForWorkerNode(ctx context.Context, k3sclient client.Client, scope *scope.VjailbreakNodeScope) (string, error) {
	vjNode := scope.VjailbreakNode
	log := scope.Logger

	// Update the VjailbreakNode status
	err := k3sclient.Status().Update(ctx, vjNode)
	if err != nil {
		return "", errors.Wrap(err, "failed to update vjailbreak node status")
	}
	imageID, err := GetImageID(ctx, k3sclient)
	if err != nil {
		return "", errors.Wrap(err, "failed to get image id")
	}

	token, err := os.ReadFile(constants.K3sTokenFileLocation)
	if err != nil {
		return "", errors.Wrap(err, "failed to read k3s token file")
	}

	masterNode, err := GetMasterK8sNode(ctx, k3sclient)
	if err != nil {
		return "", errors.Wrap(err, "failed to get master node")
	}

	creds, err := GetOpenstackCredsVjailbreakNode(ctx, k3sclient, vjNode)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack creds")
	}

	openstackClients, err := GetOpenStackClients(ctx, k3sclient, creds)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack clients")
	}

	// Get the master node's VjailbreakNode to retrieve its OpenStack UUID
	masterVjNode := vjailbreakv1alpha1.VjailbreakNode{}
	err = k3sclient.Get(ctx, types.NamespacedName{
		Namespace: constants.NamespaceMigrationSystem,
		Name:      constants.VjailbreakMasterNodeName,
	}, &masterVjNode)
	if err != nil {
		return "", errors.Wrap(err, "failed to get master vjailbreak node")
	}

	networkIDs, masterSecurityGroups, err := GetCurrentInstanceNetworkInfo()
	if err != nil {
		return "", errors.Wrap(err, "failed to get network info")
	}

	// Determine security groups: use spec value if provided, otherwise use master's
	var securityGroups []string
	if len(vjNode.Spec.OpenstackSecurityGroups) > 0 {
		securityGroups = vjNode.Spec.OpenstackSecurityGroups
		log.Info("Using security groups from spec", "securityGroups", securityGroups)
	} else {
		securityGroups = masterSecurityGroups
		log.Info("Using security groups from master node", "securityGroups", securityGroups)
	}

	// Determine volume type: use spec value if provided, otherwise get from master node
	var volumeType string
	var availabilityZone string

	if vjNode.Spec.OpenstackVolumeType != "" {
		// Use the volume type specified in the spec
		volumeType = vjNode.Spec.OpenstackVolumeType
		log.Info("Using volume type from spec", "volumeType", volumeType)

		// Still need to get availability zone from master
		_, availabilityZone, err = GetVolumeTypeAndAvailabilityZoneFromVM(ctx, k3sclient, masterVjNode.Status.OpenstackUUID, creds)
		if err != nil {
			log.Info("Failed to get availability zone from master node, using default", "error", err)
			availabilityZone = ""
		}
	} else {
		// Get both volume type and availability zone from the master node
		volumeType, availabilityZone, err = GetVolumeTypeAndAvailabilityZoneFromVM(ctx, k3sclient, masterVjNode.Status.OpenstackUUID, creds)
		if err != nil {
			log.Info("Failed to get volume type and availability zone from master node, using defaults", "error", err)
			volumeType = ""
			availabilityZone = ""
		} else {
			log.Info("Retrieved volume type and availability zone from master node", "volumeType", volumeType, "availabilityZone", availabilityZone)
		}
	}

	// Get the flavor details to determine disk size
	flavor, err := flavors.Get(ctx, openstackClients.ComputeClient, vjNode.Spec.OpenstackFlavorID).Extract()
	if err != nil {
		return "", errors.Wrap(err, "failed to get flavor details")
	}

	// Use flavor disk size, but ensure it's at least 60GB
	diskSize := flavor.Disk
	if diskSize < 60 {
		diskSize = 60
		log.Info("Flavor disk size is less than 60GB, using minimum of 60GB", "flavorDisk", flavor.Disk, "actualSize", diskSize)
	}

	// Set Nova API microversion to support volume_type in block_device_mapping_v2
	// Volume type in block device mapping requires microversion 2.67+
	openstackClients.ComputeClient.Microversion = "2.67"

	log.Info("Creating agent node with volume type", "volumeType", volumeType, "size", diskSize, "flavor", flavor.Name)

	// Create root disk from image with volume type
	rootDisk := servers.BlockDevice{
		SourceType:          servers.SourceImage,
		DestinationType:     servers.DestinationVolume,
		UUID:                imageID,
		BootIndex:           0,
		DeleteOnTermination: true,
		VolumeSize:          diskSize,
	}

	// Only set volume type if it's not empty
	if volumeType != "" {
		rootDisk.VolumeType = volumeType
	}

	// Define server creation parameters
	serverCreateOpts := servers.CreateOpts{
		Name:           vjNode.Name,
		FlavorRef:      vjNode.Spec.OpenstackFlavorID,
		Networks:       networkIDs,
		SecurityGroups: securityGroups,
		UserData: []byte(fmt.Sprintf(constants.K3sCloudInitScript,
			constants.ENVFileLocation,
			"false", GetNodeInternalIP(masterNode),
			token)),
		BlockDevice:      []servers.BlockDevice{rootDisk},
		AvailabilityZone: availabilityZone,
	}

	// Create the VM
	server, err := servers.Create(ctx, openstackClients.ComputeClient, serverCreateOpts, nil).Extract()
	if err != nil {
		return "", errors.Wrap(err, "Failed to create server")
	}

	log.Info("Server created", "ID", server.ID)
	return server.ID, nil
}

// GetOpenstackCredsVjailbreakNode retrieves OpenStack credentials for the master node
func GetOpenstackCredsVjailbreakNode(ctx context.Context, k3sclient client.Client, vjNode *vjailbreakv1alpha1.VjailbreakNode) (*vjailbreakv1alpha1.OpenstackCreds, error) {
	oscreds := &vjailbreakv1alpha1.OpenstackCreds{}
	err := k3sclient.Get(ctx, client.ObjectKey{
		Name:      vjNode.Spec.OpenstackCreds.Name,
		Namespace: constants.NamespaceMigrationSystem,
	}, oscreds)
	if err != nil && !apierrors.IsNotFound(err) {
		fmt.Printf("failed to get openstack creds associated with the vjailbreakNode. Using latest available creds : %v", err)
	}

	if err == nil {
		return oscreds, nil
	}
	// fetch the latest openstackcreds
	oscredsList := &vjailbreakv1alpha1.OpenstackCredsList{}
	err = k3sclient.List(ctx, oscredsList)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list openstack creds")
	}
	if len(oscredsList.Items) == 0 {
		return nil, errors.New("no openstack creds found")
	}
	oscreds = &oscredsList.Items[0]
	return oscreds, nil
}

// getSecurityGroupsFromMetadata fetches security groups from EC2-compatible metadata API
func getSecurityGroupsFromMetadata(client *retryablehttp.Client) ([]string, error) {
	req, err := retryablehttp.NewRequestWithContext(context.Background(), "GET",
		"http://169.254.169.254/2009-04-04/meta-data/security-groups", http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create security groups request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get security groups response")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing security groups response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read security groups response body")
	}

	// Security groups are returned as newline-separated values
	securityGroupsStr := strings.TrimSpace(string(body))
	if securityGroupsStr == "" {
		return []string{}, nil
	}

	securityGroups := strings.Split(securityGroupsStr, "\n")
	return securityGroups, nil
}

// GetCurrentInstanceNetworkInfo retrieves network and security group information for the current instance
func GetCurrentInstanceNetworkInfo() ([]servers.Network, []string, error) {
	client := retryablehttp.NewClient()
	client.RetryMax = 5
	client.Logger = nil
	networks := []servers.Network{}

	// Fetch network data
	req, err := retryablehttp.NewRequestWithContext(context.Background(), "GET",
		"http://169.254.169.254/openstack/latest/network_data.json", http.NoBody)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get response")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read response body")
	}

	var metadata OpenStackMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	for _, Network := range metadata.Networks {
		networks = append(networks, servers.Network{
			UUID: Network.NetworkID,
		})
	}

	// Fetch security groups from EC2-compatible metadata API
	securityGroups, err := getSecurityGroupsFromMetadata(client)
	if err != nil {
		// Log the error but don't fail the entire operation
		fmt.Printf("Warning: failed to get security groups: %v\n", err)
		return networks, []string{}, nil
	}

	return networks, securityGroups, nil
}

// GetOpenstackVMIP retrieves the IP address of an OpenStack VM
func GetOpenstackVMIP(ctx context.Context, k3sclient client.Client, vjNode *vjailbreakv1alpha1.VjailbreakNode, uuid string) (string, error) {
	creds, err := GetOpenstackCredsVjailbreakNode(ctx, k3sclient, vjNode)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack creds")
	}
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, creds)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack clients")
	}

	// Fetch the VM details
	server, err := servers.Get(ctx, openstackClients.ComputeClient, uuid).Extract()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get server details")
	}

	// Extract IP addresses
	for _, addresses := range server.Addresses {
		addrs, ok := addresses.([]any)
		if !ok {
			return "", fmt.Errorf("addresses is not of type []any")
		}
		for _, addr := range addrs {
			ipInfo, ok := addr.(map[string]any)
			if !ok {
				continue
			}
			addrStr, ok := ipInfo["addr"].(string)
			if !ok {
				continue
			}
			return addrStr, nil
		}
	}
	return "", errors.New("failed to get vm ip")
}

// GetOpenstackVMStatus retrieves the status and IP of an OpenStack VM
// Returns: status (string), ip (string), error
func GetOpenstackVMStatus(ctx context.Context, k3sclient client.Client, vjNode *vjailbreakv1alpha1.VjailbreakNode, uuid string) (string, string, error) {
	creds, err := GetOpenstackCredsVjailbreakNode(ctx, k3sclient, vjNode)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get openstack creds")
	}
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, creds)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get openstack clients")
	}

	// Fetch the VM details
	server, err := servers.Get(ctx, openstackClients.ComputeClient, uuid).Extract()
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to get server details")
	}

	// Get VM status
	vmStatus := server.Status

	// Extract IP addresses
	var vmIP string
	for _, addresses := range server.Addresses {
		addrs, ok := addresses.([]any)
		if !ok {
			continue
		}
		for _, addr := range addrs {
			ipInfo, ok := addr.(map[string]any)
			if !ok {
				continue
			}
			addrStr, ok := ipInfo["addr"].(string)
			if ok && addrStr != "" {
				vmIP = addrStr
				break
			}
		}
		if vmIP != "" {
			break
		}
	}

	return vmStatus, vmIP, nil
}

// GetImageIDFromVM retrieves the image ID from a virtual machine using its UUID
func GetImageIDFromVM(ctx context.Context, k3sclient client.Client, uuid string, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) (string, error) {
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, openstackcreds)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack clients")
	}

	// Fetch the VM details
	server, err := servers.Get(ctx, openstackClients.ComputeClient, uuid).Extract()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get server details")
	}

	if server.Image["id"] != nil {
		fmt.Println("Image ID found", "Image ID", server.Image["id"])
	} else {
		imageID, err := GetImageIDOfVMBootFromVolume(ctx, uuid, k3sclient, openstackcreds)
		if err != nil {
			return "", errors.Wrap(err, "Failed to get image ID from VM or volume")
		}
		return imageID, nil
	}

	if imageID, ok := server.Image["id"].(string); ok {
		return imageID, nil
	}
	return "", fmt.Errorf("failed to assert image ID as string")
}

// GetImageIDOfVMBootFromVolume returns the ID of the image used to create the volume
func GetImageIDOfVMBootFromVolume(ctx context.Context, uuid string, k3sclient client.Client, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) (string, error) {
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, openstackcreds)
	if err != nil {
		return "", errors.Wrap(err, "failed to get OpenStack clients")
	}

	// Fetch the VM details
	server, err := servers.Get(ctx, openstackClients.ComputeClient, uuid).Extract()
	if err != nil {
		return "", errors.Wrap(err, "failed to get server details")
	}

	// Get attached volumes on that server
	attachedVolumes := server.AttachedVolumes

	// Check if the root volume is an image
	for _, volume := range attachedVolumes {
		// Get volume details
		volume, err := volumes.Get(ctx, openstackClients.BlockStorageClient, volume.ID).Extract()
		if err != nil {
			return "", errors.Wrap(err, "failed to get volume details")
		}
		fmt.Println("Volume metadata", volume.VolumeImageMetadata)
		imageID := volume.VolumeImageMetadata["image_id"]
		if imageID != "" {
			return imageID, nil
		}
	}
	return "", fmt.Errorf("no image found for the volume")
}

// GetVolumeTypeFromVM retrieves the volume type from a virtual machine using its UUID
func GetVolumeTypeFromVM(ctx context.Context, k3sclient client.Client, uuid string, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) (string, error) {
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, openstackcreds)
	if err != nil {
		return "", errors.Wrap(err, "failed to get OpenStack clients")
	}

	// Fetch the VM details
	server, err := servers.Get(ctx, openstackClients.ComputeClient, uuid).Extract()
	if err != nil {
		return "", errors.Wrap(err, "failed to get server details")
	}

	// Get attached volumes on that server
	attachedVolumes := server.AttachedVolumes

	// Get the root volume's type (typically the first volume or boot volume)
	for _, attachedVol := range attachedVolumes {
		// Get volume details
		volume, err := volumes.Get(ctx, openstackClients.BlockStorageClient, attachedVol.ID).Extract()
		if err != nil {
			return "", errors.Wrap(err, "failed to get volume details")
		}
		// Return the volume type if found
		if volume.VolumeType != "" {
			return volume.VolumeType, nil
		}
	}
	return "", fmt.Errorf("no volume type found for the VM")
}

// GetAvailabilityZoneFromVM retrieves the availability zone from a virtual machine using its UUID
func GetAvailabilityZoneFromVM(ctx context.Context, k3sclient client.Client, uuid string, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) (string, error) {
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, openstackcreds)
	if err != nil {
		return "", errors.Wrap(err, "failed to get OpenStack clients")
	}

	// Fetch the VM details
	server, err := servers.Get(ctx, openstackClients.ComputeClient, uuid).Extract()
	if err != nil {
		return "", errors.Wrap(err, "failed to get server details")
	}

	// Return the availability zone
	return server.AvailabilityZone, nil
}

// GetVolumeTypeAndAvailabilityZoneFromVM retrieves both volume type and availability zone from a VM in a single call
func GetVolumeTypeAndAvailabilityZoneFromVM(ctx context.Context, k3sclient client.Client, uuid string, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) (volumeType string, availabilityZone string, err error) {
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, openstackcreds)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get OpenStack clients")
	}

	// Fetch the VM details
	server, err := servers.Get(ctx, openstackClients.ComputeClient, uuid).Extract()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get server details")
	}

	// Get the availability zone from the server
	availabilityZone = server.AvailabilityZone

	// Get attached volumes on that server
	attachedVolumes := server.AttachedVolumes

	// Get the root volume's type (typically the first volume or boot volume)
	for _, attachedVol := range attachedVolumes {
		// Get volume details
		volume, err := volumes.Get(ctx, openstackClients.BlockStorageClient, attachedVol.ID).Extract()
		if err != nil {
			return "", availabilityZone, errors.Wrap(err, "failed to get volume details")
		}
		// Return the volume type if found
		if volume.VolumeType != "" {
			volumeType = volume.VolumeType
			break
		}
	}

	return volumeType, availabilityZone, nil
}

// ListAllFlavors retrieves a list of all available OpenStack flavors
func ListAllFlavors(ctx context.Context, k3sclient client.Client, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) ([]flavors.Flavor, error) {
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, openstackcreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get openstack clients")
	}

	// List flavors
	allPages, err := flavors.ListDetail(openstackClients.ComputeClient, nil).AllPages(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list flavors")
	}

	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return nil, err
	}

	// Ensure ExtraSpecs is never nil to satisfy CRD validation requirements
	for i := range flavorList {
		if flavorList[i].ExtraSpecs == nil {
			flavorList[i].ExtraSpecs = make(map[string]string)
		}

		// Fetch flavor-specific extra_specs from OpenStack/PCD to retain GPU traits/aliases
		extraSpecs, extraErr := flavors.ListExtraSpecs(ctx, openstackClients.ComputeClient, flavorList[i].ID).Extract()
		if extraErr != nil {
			return nil, errors.Wrapf(extraErr, "failed to list extra specs for flavor %q", flavorList[i].Name)
		}
		for k, v := range extraSpecs {
			flavorList[i].ExtraSpecs[k] = v
		}
	}

	return flavorList, nil
}

// DeleteOpenstackVM deletes an OpenStack virtual machine by its UUID
func DeleteOpenstackVM(ctx context.Context, uuid string, k3sclient client.Client, vjNode *vjailbreakv1alpha1.VjailbreakNode) error {
	creds, err := GetOpenstackCredsVjailbreakNode(ctx, k3sclient, vjNode)
	if err != nil {
		return errors.Wrap(err, "failed to get openstack creds")
	}
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, creds)
	if err != nil {
		return errors.Wrap(err, "failed to get openstack clients")
	}

	// delete the VM
	err = servers.Delete(ctx, openstackClients.ComputeClient, uuid).ExtractErr()
	if err != nil && !strings.Contains(err.Error(), "404") {
		return errors.Wrap(err, "Failed to delete server")
	}
	return nil
}

// GetImageID retrieves the image ID from the Kubernetes client
func GetImageID(ctx context.Context, k3sclient client.Client) (string, error) {
	vjNode := vjailbreakv1alpha1.VjailbreakNode{}
	// Get the image ID from the vjailbreak master node
	err := k3sclient.Get(ctx, types.NamespacedName{
		Namespace: constants.NamespaceMigrationSystem,
		Name:      constants.VjailbreakMasterNodeName,
	}, &vjNode)
	if err != nil {
		return "", errors.Wrap(err, "failed to get vjailbreak node")
	}
	return vjNode.Spec.OpenstackImageID, nil
}

// GetOpenstackVMByName retrieves an OpenStack VM's UUID by its name
func GetOpenstackVMByName(ctx context.Context, name string, k3sclient client.Client, vjNode *vjailbreakv1alpha1.VjailbreakNode) (string, error) {
	creds, err := GetOpenstackCredsVjailbreakNode(ctx, k3sclient, vjNode)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack creds")
	}
	openstackClients, err := GetOpenStackClients(ctx, k3sclient, creds)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack clients")
	}

	listOpts := servers.ListOpts{Name: name}
	allPages, err := servers.List(openstackClients.ComputeClient, listOpts).AllPages(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to list servers")
	}

	allServers, err := servers.ExtractServers(allPages)
	if err != nil || len(allServers) == 0 {
		return "", errors.Wrap(err, "failed to extract servers")
	}

	vmID := allServers[0].ID
	return vmID, nil
}

// ReadFileContent reads and returns the content of a file at the given path
func ReadFileContent(filePath string) ([]byte, error) {
	// Validate file path
	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, "/") {
		return nil, fmt.Errorf("invalid file path: must be absolute path")
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	return data, nil
}

// GetActiveMigrations retrieves a list of active migrations for a given node
func GetActiveMigrations(ctx context.Context, nodeName string, k3sclient client.Client) ([]string, error) {
	migrationList := &vjailbreakv1alpha1.MigrationList{}
	err := k3sclient.List(ctx, migrationList)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list migrations")
	}

	ignorePhases := []vjailbreakv1alpha1.VMMigrationPhase{vjailbreakv1alpha1.VMMigrationPhasePending,
		vjailbreakv1alpha1.VMMigrationPhaseFailed,
		vjailbreakv1alpha1.VMMigrationPhaseSucceeded,
		vjailbreakv1alpha1.VMMigrationPhaseUnknown,
	}

	var activeMigrations []string
	for i := range migrationList.Items {
		migration := &migrationList.Items[i]
		if migration.Status.AgentName == nodeName && !slices.Contains(ignorePhases, migration.Status.Phase) {
			activeMigrations = append(activeMigrations,
				migration.Name)
		}
	}
	return activeMigrations, nil
}

// GetInclusterClient creates and returns a Kubernetes in-cluster client
func GetInclusterClient() (client.Client, error) {
	// Create a direct Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get in-cluster config")
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(vjailbreakv1alpha1.AddToScheme(scheme))
	clientset, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get in-cluster config")
	}

	return clientset, err
}

// GetNodeByName returns the node object by name
func GetNodeByName(ctx context.Context, k3sclient client.Client, nodeName string) (*corev1.Node, error) {
	node := &corev1.Node{}
	err := k3sclient.Get(ctx, client.ObjectKey{
		Name: nodeName,
	}, node)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node")
	}
	return node, nil
}

// DeleteNodeByName deletes a Kubernetes node by its name
func DeleteNodeByName(ctx context.Context, k3sclient client.Client, nodeName string) error {
	node := &corev1.Node{}
	err := k3sclient.Get(ctx, client.ObjectKey{
		Name: nodeName,
	}, node)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get node")
	}
	err = k3sclient.Delete(ctx, node)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete node")
	}
	return nil
}

// GetVMMigration retrieves a Migration resource for a specific VM in a rolling migration plan.
// It returns the Migration resource associated with the VM or an error if not found.
func GetVMMigration(ctx context.Context, k3sclient client.Client, vmName string, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.Migration, error) {
	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, k3sclient, rollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware credentials")
	}
	vmk8sName, err := GetK8sCompatibleVMWareObjectName(vmName, vmwarecreds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vm name")
	}
	migration := &vjailbreakv1alpha1.Migration{}
	err = k3sclient.Get(ctx, client.ObjectKey{
		Name:      MigrationNameFromVMName(vmk8sName),
		Namespace: rollingMigrationPlan.Namespace,
	}, migration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vm migration")
	}
	return migration, nil
}

// ReconcileVMStatusAndIP handles VM status checking and IP population for a VjailbreakNode
func ReconcileVMStatusAndIP(ctx context.Context, k8sClient client.Client, vjNode *vjailbreakv1alpha1.VjailbreakNode, uuid string) (bool, error) {
	// Get VM status and IP from OpenStack
	vmStatus, vmIP, err := GetOpenstackVMStatus(ctx, k8sClient, vjNode, uuid)
	if err != nil {
		vjNode.Status.Phase = constants.VjailbreakNodePhaseError
		if updateErr := k8sClient.Status().Update(ctx, vjNode); updateErr != nil {
			return false, errors.Wrap(updateErr, "failed to update node status to error")
		}
		return false, errors.Wrap(err, "failed to get vm status from openstack")
	}

	// Handle different VM states
	switch vmStatus {
	case "ERROR":
		vjNode.Status.Phase = constants.VjailbreakNodePhaseError
		if updateErr := k8sClient.Status().Update(ctx, vjNode); updateErr != nil {
			return false, errors.Wrap(updateErr, "failed to update node status to error")
		}
		return false, errors.New("VM is in ERROR state in OpenStack")
	case "ACTIVE":
		// VM is active, proceed with IP assignment
		vjNode.Status.Phase = constants.VjailbreakNodePhaseVMCreated
		if vmIP == "" {
			// IP not yet available, caller should retry
			return false, nil
		}
		vjNode.Status.VMIP = vmIP
	default:
		// VM is still building or in another transitional state
		vjNode.Status.Phase = constants.VjailbreakNodePhaseVMCreating
		if updateErr := k8sClient.Status().Update(ctx, vjNode); updateErr != nil {
			return false, errors.Wrap(updateErr, "failed to update node status")
		}
		return false, nil
	}

	// Set UUID if not already set
	if vjNode.Status.OpenstackUUID == "" {
		vjNode.Status.OpenstackUUID = uuid
	}

	// Update the VjailbreakNode status
	err = k8sClient.Status().Update(ctx, vjNode)
	if err != nil {
		return false, errors.Wrap(err, "failed to update vjailbreak node status")
	}

	// Return true if IP was set
	return vjNode.Status.VMIP != "", nil
}

// ReconcileK8sNodeStatus checks Kubernetes node status and updates VjailbreakNode phase accordingly
func ReconcileK8sNodeStatus(ctx context.Context, k8sClient client.Client, vjNode *vjailbreakv1alpha1.VjailbreakNode) (bool, error) {
	node, err := GetNodeByName(ctx, k8sClient, vjNode.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Keep phase as VMCreated while waiting for K8s node
			if vjNode.Status.Phase != constants.VjailbreakNodePhaseVMCreated {
				vjNode.Status.Phase = constants.VjailbreakNodePhaseVMCreated
				if updateErr := k8sClient.Status().Update(ctx, vjNode); updateErr != nil {
					return false, errors.Wrap(updateErr, "failed to update node status")
				}
			}
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get node by name")
	}

	// Check if node is ready
	nodeReady := IsNodeReady(node)

	// Update phase based on node readiness
	if nodeReady {
		vjNode.Status.Phase = constants.VjailbreakNodePhaseNodeReady
	} else if vjNode.Status.Phase != constants.VjailbreakNodePhaseVMCreated {
		vjNode.Status.Phase = constants.VjailbreakNodePhaseVMCreated
	}

	// Update the VjailbreakNode status
	err = k8sClient.Status().Update(ctx, vjNode)
	if err != nil {
		return false, errors.Wrap(err, "failed to update vjailbreak node status")
	}

	return nodeReady, nil
}

// IsNodeReady checks if a Kubernetes node is in Ready state
func IsNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
