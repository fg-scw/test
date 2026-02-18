package storage

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// VendorConfig holds vendor-specific configuration
type VendorConfig struct {
	// NAA prefix for this vendor (e.g., "624a9370" for Pure, "60a98000" for NetApp)
	NAAPrefix string
	// Name of the vendor
	Name string
}

// BaseStorageProvider provides common functionality for storage providers
type BaseStorageProvider struct {
	Client      *http.Client
	BaseURL     string
	Username    string
	Password    string
	AccessInfo  StorageAccessInfo
	IsConnected bool
	Config      VendorConfig
}

// InitHTTPClient initializes the HTTP client with SSL configuration
func (b *BaseStorageProvider) InitHTTPClient(skipSSLVerify bool) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipSSLVerify,
		},
	}
	b.Client = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// DoRequest performs an HTTP request with basic auth
func (b *BaseStorageProvider) DoRequest(ctx context.Context, method, endpoint string, body io.Reader) ([]byte, error) {
	url := b.BaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(b.Username, b.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// DoRequestJSON performs an HTTP request and unmarshals JSON response
func (b *BaseStorageProvider) DoRequestJSON(ctx context.Context, method, endpoint string, body io.Reader, result interface{}) error {
	data, err := b.DoRequest(ctx, method, endpoint, body)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// BuildNAA constructs a NAA identifier from a serial number
func (b *BaseStorageProvider) BuildNAA(serialNumber string) string {
	return fmt.Sprintf("naa.%s%s", b.Config.NAAPrefix, strings.ToLower(serialNumber))
}

// ExtractSerialFromNAA extracts the serial number from a NAA identifier
func (b *BaseStorageProvider) ExtractSerialFromNAA(naaID string) (string, error) {
	prefix := "naa." + b.Config.NAAPrefix
	if !strings.HasPrefix(naaID, prefix) {
		return "", fmt.Errorf("NAA ID %s is not from %s (expected prefix: %s)", naaID, b.Config.Name, prefix)
	}

	serial := strings.TrimPrefix(naaID, prefix)
	return strings.ToUpper(serial), nil
}

// IsValidNAA checks if the NAA belongs to this vendor
func (b *BaseStorageProvider) IsValidNAA(naaID string) bool {
	return strings.HasPrefix(naaID, "naa."+b.Config.NAAPrefix)
}

// GetAllVolumeNAAs retrieves NAA identifiers for all volumes
// This is a common implementation that uses ListAllVolumes
func (b *BaseStorageProvider) GetAllVolumeNAAs(listFn func() ([]VolumeInfo, error)) ([]string, error) {
	volumes, err := listFn()
	if err != nil {
		return nil, err
	}

	var naaIdentifiers []string
	for _, v := range volumes {
		naaIdentifiers = append(naaIdentifiers, v.NAA)
	}

	return naaIdentifiers, nil
}

// GetVolumeFromNAACommon finds a volume by NAA using the provided list function
func (b *BaseStorageProvider) GetVolumeFromNAACommon(naaID string, listFn func() ([]VolumeInfo, error), getVolumeFn func(name string) (Volume, error)) (Volume, error) {
	serial, err := b.ExtractSerialFromNAA(naaID)
	if err != nil {
		return Volume{}, err
	}

	volumes, err := listFn()
	if err != nil {
		return Volume{}, fmt.Errorf("failed to list volumes: %w", err)
	}

	for _, v := range volumes {
		// Extract serial from the volume's NAA and compare
		volSerial, err := b.ExtractSerialFromNAA(v.NAA)
		if err != nil {
			klog.Warningf("Invalid NAA format for volume %s: %v", v.Name, err)
			continue
		}
		if volSerial == serial {
			klog.Infof("Found %s volume %s matching NAA %s", b.Config.Name, v.Name, naaID)
			vol, err := getVolumeFn(v.Name)
			if err != nil {
				return Volume{}, err
			}
			return vol, nil
		}
	}

	return Volume{}, fmt.Errorf("no %s volume found with NAA %s (serial: %s)", b.Config.Name, naaID, serial)
}

// SetConnected sets the connection state
func (b *BaseStorageProvider) SetConnected(connected bool) {
	b.IsConnected = connected
}

// GetConnected returns the connection state
func (b *BaseStorageProvider) GetConnected() bool {
	return b.IsConnected
}

// Helper functions

// ContainsIgnoreCase checks if a slice contains a string (case-insensitive)
func ContainsIgnoreCase(slice []string, item string) bool {
	itemLower := strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == itemLower {
			return true
		}
	}
	return false
}

// SliceContains checks if a slice contains a string (case-sensitive)
func SliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
