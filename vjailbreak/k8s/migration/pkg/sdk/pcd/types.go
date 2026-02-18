// Package du provides utilities and types for interacting with Platform9 Distributed Cloud services.
// It includes configuration types and helper functions for Platform9 Distributed Cloud operations.
package du

// Info contains connection information for a Platform9 Distributed Cloud instance,
// including the URL and whether to skip TLS certificate verification.
type Info struct {
	URL      string
	Insecure bool
}
