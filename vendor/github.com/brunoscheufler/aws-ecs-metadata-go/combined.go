package metadata

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// Will retrieve the task metadata for compatibility
func Get(ctx context.Context, client *http.Client) (interface{}, error) {
	return GetTask(ctx, client)
}

// Will retrieve the task metadata
// for your current Fargate environment (either V3 or V4)
// based on the environment variables that are present
func GetTask(ctx context.Context, client *http.Client) (interface{}, error) {
	// If the ECS Metadata URI for v4 is set,
	// use this. When running on platform version 4,
	// v3 might also still be set, though we prioritize the newer format
	isV4 := os.Getenv(ecsMetadataUriEnvV4) != ""
	if isV4 {
		return GetTaskV4(ctx, client)
	}

	// If the Metadata URI for v4 wasn't set,
	// check for v3
	isV3 := os.Getenv(ecsMetadataUriEnvV3) != ""
	if isV3 {
		return GetTaskV3(ctx, client)
	}

	return nil, fmt.Errorf("could not resolve ECS Task metadata")
}

// Will retrieve the container metadata
// for your current Fargate environment (either V3 or V4)
// based on the environment variables that are present
func GetContainer(ctx context.Context, client *http.Client) (interface{}, error) {
	// If the ECS Metadata URI for v4 is set,
	// use this. When running on platform version 4,
	// v3 might also still be set, though we prioritize the newer format
	isV4 := os.Getenv(ecsMetadataUriEnvV4) != ""
	if isV4 {
		return GetContainerV4(ctx, client)
	}

	// If the Metadata URI for v4 wasn't set,
	// check for v3
	isV3 := os.Getenv(ecsMetadataUriEnvV3) != ""
	if isV3 {
		return GetContainerV3(ctx, client)
	}

	return nil, fmt.Errorf("could not resolve ECS Task metadata")
}

// Will check whether the environment
// has available metadata URIs to use for fetching
func Has() bool {
	if os.Getenv(ecsMetadataUriEnvV3) != "" {
		return true
	}

	if os.Getenv(ecsMetadataUriEnvV4) != "" {
		return true
	}

	return false
}
