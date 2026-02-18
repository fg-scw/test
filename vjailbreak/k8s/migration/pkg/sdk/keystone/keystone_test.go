package keystone_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/keystone"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/testutils"
)

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

var serviceInfo = `{"services": [{"description": "Links to region specific resources hosted on a DU","name": "regioninfo", "id": "6d30c85c033247548d6d93b0056b266b", "type": "regioninfo", "enabled": true, "links": { "self": "https://example.platform9.horse/keystone/v3/services/6d30c85c033247548d6d93b0056b266b"}}], "links": { "next": null, "self": "https://example.platform9.horse/keystone/v3/services?type=regionInfo", "previous": null}}`

var serviceIDExpected = "6d30c85c033247548d6d93b0056b266b"

// Tests the API to fetch cluster DuFqdn.
func TestGetServiceID(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test request parameters
		testutils.Equals(t, req.URL.String(), "http://example.com?type=regionInfo")
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: io.NopCloser(bytes.NewBufferString(serviceInfo)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	sAPI := keystone.ServiceManagerAPI{client, "http://example.com", "token"}
	serviceIDActual, err := sAPI.GetServiceIDAPI("regionInfo")
	testutils.Ok(t, err)
	testutils.Equals(t, serviceIDExpected, serviceIDActual)
}

var endpointInfo = `{
  "endpoints": [
    {
      "id": "08732d8b2c29499883d96b4e63a6abd0",
      "interface": "public",
      "region_id": "region1",
      "serviceID": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example-region1.platform9.horse/links/",
      "enabled": true,
      "region": "region1",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/08732d8b2c29499883d96b4e63a6abd0"
      }
    },
    {
      "id": "36b1a102daa84360b7dc55c09b85b6fd",
      "interface": "internal",
      "region_id": "region1",
      "serviceID": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example-region1.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region1",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/36b1a102daa84360b7dc55c09b85b6fd"
      }
    },
    {
      "id": "3d8cbf02660648d6bfd3b18697372488",
      "interface": "admin",
      "region_id": "region2",
      "serviceID": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region2",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/3d8cbf02660648d6bfd3b18697372488"
      }
    },
    {
      "id": "4310d6bd9b34451b8d04ec1992ecbe70",
      "interface": "internal",
      "region_id": "region2",
      "serviceID": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region2",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/4310d6bd9b34451b8d04ec1992ecbe70"
      }
    },
    {
      "id": "d55e08a83f2141c092d2d0e339bf501e",
      "interface": "public",
      "region_id": "region2",
      "serviceID": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example.platform9.horse/links/",
      "enabled": true,
      "region": "region2",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/d55e08a83f2141c092d2d0e339bf501e"
      }
    },
    {
      "id": "ff1634c5e83042f49f6cca837c254aa3",
      "interface": "admin",
      "region_id": "region1",
      "serviceID": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example-region1.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region1",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/ff1634c5e83042f49f6cca837c254aa3"
      }
    }
  ],
  "links": {
    "next": null,
    "self": "https://example-region1.platform9.horse/keystone/v3/endpoints?serviceID=6d30c85c033247548d6d93b0056b266b",
    "previous": null
  }
}`

var region1EndpointExpected = "example-region1.platform9.horse"
var region2EndpointExpected = "example.platform9.horse"

// Tests the API to fetch cluster URL.
func TestGetEndpointForRegion(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test request parameters
		testutils.Equals(t, req.URL.String(), "http://example.com?serviceID=6d30c85c033247548d6d93b0056b266b")
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: io.NopCloser(bytes.NewBufferString(endpointInfo)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	eAPI := keystone.EndpointManagerAPI{client, "http://example.com", "token"}

	// Test for region1
	endpointActual, err := eAPI.GetEndpointForRegionAPI("region1", "6d30c85c033247548d6d93b0056b266b")
	testutils.Ok(t, err)
	testutils.Equals(t, region1EndpointExpected, endpointActual)

	// Test for region2
	endpointActual, err = eAPI.GetEndpointForRegionAPI("region2", "6d30c85c033247548d6d93b0056b266b")
	testutils.Ok(t, err)
	testutils.Equals(t, region2EndpointExpected, endpointActual)
}
