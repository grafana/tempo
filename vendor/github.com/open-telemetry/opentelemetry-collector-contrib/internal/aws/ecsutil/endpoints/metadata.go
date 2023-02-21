// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package endpoints // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil/endpoints"

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	TaskMetadataEndpointV3EnvVar = "ECS_CONTAINER_METADATA_URI"
	TaskMetadataEndpointV4EnvVar = "ECS_CONTAINER_METADATA_URI_V4"

	TaskMetadataPath      = "/task"
	ContainerMetadataPath = ""
)

// ErrNoTaskMetadataEndpointDetected is a reserved error type to distinguish between incompatible environments
// and other error scenarios
type ErrNoTaskMetadataEndpointDetected struct {
	error
	MissingVersion int
}

// GetTMEV3FromEnv will return a validated task metadata endpoint as obtained by the v3 env var, if any.
func GetTMEV3FromEnv() (endpoint *url.URL, err error) {
	endpoint, err = validateEndpoint(os.Getenv(TaskMetadataEndpointV3EnvVar))
	if err != nil {
		endpoint = nil
		err = ErrNoTaskMetadataEndpointDetected{
			fmt.Errorf("no valid endpoint for environment variable %s: %w", TaskMetadataEndpointV3EnvVar, err), 3,
		}
	}
	return
}

// GetTMEV4FromEnv will return a validated task metadata endpoint as obtained by the v4 env var, if any.
func GetTMEV4FromEnv() (endpoint *url.URL, err error) {
	endpoint, err = validateEndpoint(os.Getenv(TaskMetadataEndpointV4EnvVar))
	if err != nil {
		endpoint = nil
		err = ErrNoTaskMetadataEndpointDetected{
			fmt.Errorf("no valid endpoint for environment variable %s: %w", TaskMetadataEndpointV4EnvVar, err), 4,
		}
	}
	return
}

// GetTMEFromEnv will return the first available task metadata endpoint for the v4 or v3 env var in that order.
func GetTMEFromEnv() (endpoint *url.URL, err error) {
	if endpoint, err = GetTMEV4FromEnv(); err != nil {
		endpoint, err = GetTMEV3FromEnv()
	}
	return
}

func validateEndpoint(candidate string) (endpoint *url.URL, err error) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		err = fmt.Errorf("endpoint is empty")
		return
	}

	endpoint, err = url.ParseRequestURI(candidate)
	return
}
