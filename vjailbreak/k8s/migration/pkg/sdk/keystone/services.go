// Copyright © 2021 The Platform9 Systems Inc.

package keystone

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// ServiceManagerAPI encapsulates the functionality to interact with OpenStack service catalog
// through the Keystone API. It provides methods to retrieve service information and IDs.
type ServiceManagerAPI struct {
	Client  *http.Client
	BaseURL string
	Token   string
}

// ServicesInfo represents the response structure from the Keystone API
// when querying for service information. It contains details about available services
// including their IDs, types, and names.
type ServicesInfo struct {
	Services []struct {
		Description string `json:"description"`
		Links       struct {
			Self string `json:"self"`
		} `json:"links"`
		Enabled bool   `json:"enabled"`
		Type    string `json:"type"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	} `json:"services"`
	Links struct {
		Self     string      `json:"self"`
		Previous interface{} `json:"previous"`
		Next     interface{} `json:"next"`
	} `json:"links"`
}

// GetServiceID retrieves the service ID for a specified service name from the Keystone service catalog.
// It uses the provided authentication information to query the Keystone API.
func GetServiceID(
	url string, // DU URL
	auth AuthInfo, // Auth info
	name string, // Service name
) (string, error) {
	zap.S().Debug("Fetching service ID for service: ", name)

	// Form the URL
	url = fmt.Sprintf("%s/keystone/v3/services", url)

	// Generate the http client object
	client := &http.Client{}

	// Create the context to invoke the service manager API.
	sAPI := ServiceManagerAPI{client, url, auth.Token}

	// Invoke the actual "get services" API.
	ID, err := sAPI.GetServiceIDAPI(name)
	if err != nil {
		return "", err
	}

	zap.S().Debug("service ID : ", ID)
	return ID, nil
}

// GetServiceIDAPI implements the specific API call to retrieve a service ID by name.
// It handles the HTTP request construction, execution, and response parsing.
func (sAPI *ServiceManagerAPI) GetServiceIDAPI(
	name string,
) (string, error) {
	zap.S().Debug("Fetching service ID for ", name)
	req, err := http.NewRequest("GET", sAPI.BaseURL, nil)
	if err != nil {
		zap.S().Errorf("Failed to create request for service ID for %s, Error: %s", name, err)
		return "", fmt.Errorf("failed to create request for service ID for %s, Error: %s", name, err)
	}

	// Add keystone token in the header.
	req.Header.Add("X-Auth-Token", sAPI.Token)

	// Add the query parameter "type"
	q := req.URL.Query()
	q.Add("type", name)
	req.URL.RawQuery = q.Encode()

	resp, err := sAPI.Client.Do(req)
	if err != nil {
		zap.S().Errorf("Failed to fetch service information for service %s, Error: %s", name, err)
		return "", fmt.Errorf("failed to fetch service information for service %s, Error: %s", name, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	serviceInfo := ServicesInfo{}
	// Response is received as slice of services.
	err = json.NewDecoder(resp.Body).Decode(&serviceInfo)
	if err != nil {
		zap.S().Errorf("Failed to decode service information, Error: %s", err)
		return "", fmt.Errorf("failed to decode service information, Error: %s", err)
	}

	// There is supposed to be only one service per name.
	// Pick the ID from the first instance in the slice.
	serviceID := serviceInfo.Services[0].ID
	zap.S().Debug("service ID : .", serviceID)
	return serviceID, nil
}
