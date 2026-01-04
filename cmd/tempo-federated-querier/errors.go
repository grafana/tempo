package main

import "fmt"

// Error definitions for the federated querier
var (
	errNoInstances = fmt.Errorf("at least one tempo instance must be configured")
)

func errInstanceEndpointRequired(index int) error {
	return fmt.Errorf("instance %d: endpoint is required", index)
}
