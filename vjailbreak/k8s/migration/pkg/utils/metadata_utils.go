package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/pkg/errors"
)

// InstanceMetadata contains metadata about the current instance.
type InstanceMetadata struct {
	UUID string `json:"uuid"`
}

// Package-level variables to implement a thread-safe cache.
var (
	// cachedMetadata holds the metadata once it's been successfully fetched.
	cachedMetadata *InstanceMetadata
	// metadataMutex protects access to cachedMetadata.
	metadataMutex sync.RWMutex
)

// GetCurrentInstanceMetadata retrieves metadata about the current instance from the OpenStack metadata service
func GetCurrentInstanceMetadata() (*InstanceMetadata, error) {
	// Step 1. Path with a read lock
	// First Check if the data is already cached. This readlocky allows multiple
	// Goroutines to read the cached data concurrently.

	metadataMutex.RLock()
	if cachedMetadata != nil {
		// If cached, return it immediately.
		defer metadataMutex.RUnlock()
		return cachedMetadata, nil
	}

	// Release the read lock before we get the Write lock.
	metadataMutex.RUnlock()

	// Step 2. Path with write lock
	// Acquire a write lock to fetch and cache the metadata.
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	// Check again if another goroutine has already fetched the metadata.

	if cachedMetadata != nil {
		return cachedMetadata, nil
	}

	// Step 4: Fetch metadata from the OpenStack metadata service

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "http://169.254.169.254/openstack/latest/meta_data.json", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metadata request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch instance metadata")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("[WARN] Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from metadata service: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read metadata response")
	}

	var metadata InstanceMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, errors.Wrap(err, "failed to parse metadata")
	}

	if metadata.UUID == "" {
		return nil, errors.New("instance UUID not found in metadata")
	}
	cachedMetadata = &metadata
	// Step 5: Return the fetched metadata

	return cachedMetadata, nil
}

// VerifyCredentialsMatchCurrentEnvironment checks if the provided credentials can access the current instance
func VerifyCredentialsMatchCurrentEnvironment(providerClient *gophercloud.ProviderClient, regionName string) (bool, error) {
	// Get current instance metadata
	metadata, err := GetCurrentInstanceMetadata()
	if err != nil {
		return false, fmt.Errorf("unable to get current instance metadata: %w. "+
			"Please ensure this is running on an OpenStack instance with metadata service enabled", err)
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region: regionName,
	})

	if err != nil {
		return false, fmt.Errorf("failed to create OpenStack compute client: %w", err)
	}
	_, err = servers.Get(context.TODO(), computeClient, metadata.UUID).Extract()
	if err != nil {
		if strings.Contains(err.Error(), "Resource not found") ||
			strings.Contains(err.Error(), "No server with a name or ID") {
			return false, errors.Wrap(err, "wrong OpenStack environment. Use credentials from this environment")
		}
		return false, fmt.Errorf("failed to verify instance access: %w. "+
			"Please check if the provided credentials have compute:get_server permission", err)
	}
	return true, nil
}
