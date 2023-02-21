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

package ecsutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil"

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil/endpoints"
)

type MetadataProvider interface {
	FetchTaskMetadata() (*TaskMetadata, error)
	FetchContainerMetadata() (*ContainerMetadata, error)
}

type ecsMetadataProviderImpl struct {
	logger *zap.Logger
	client RestClient
}

var _ MetadataProvider = &ecsMetadataProviderImpl{}

func NewTaskMetadataProvider(client RestClient, logger *zap.Logger) MetadataProvider {
	return &ecsMetadataProviderImpl{
		client: client,
		logger: logger,
	}
}

func NewDetectedTaskMetadataProvider(set component.TelemetrySettings) (MetadataProvider, error) {
	endpoint, err := endpoints.GetTMEFromEnv()
	if err != nil {
		return nil, err
	} else if endpoint == nil {
		return nil, fmt.Errorf("unable to detect task metadata endpoint")
	}

	clientSettings := confighttp.HTTPClientSettings{}
	client, err := NewRestClient(*endpoint, clientSettings, set)
	if err != nil {
		return nil, err
	}

	return &ecsMetadataProviderImpl{
		client: client,
		logger: set.Logger,
	}, nil
}

// FetchTaskMetadata retrieves the metadata for a task running on Amazon ECS
func (md *ecsMetadataProviderImpl) FetchTaskMetadata() (*TaskMetadata, error) {
	resp, err := md.client.GetResponse(endpoints.TaskMetadataPath)
	if err != nil {
		return nil, err
	}

	taskMetadata := &TaskMetadata{}

	err = json.NewDecoder(bytes.NewReader(resp)).Decode(taskMetadata)

	if err != nil {
		return nil, fmt.Errorf("encountered unexpected error reading response from ECS Task Metadata Endpoint: %w", err)
	}

	return taskMetadata, nil
}

// FetchContainerMetadata retrieves the metadata for the Amazon ECS Container the collector is running on
func (md *ecsMetadataProviderImpl) FetchContainerMetadata() (*ContainerMetadata, error) {
	resp, err := md.client.GetResponse(endpoints.ContainerMetadataPath)
	if err != nil {
		return nil, err
	}

	containerMetadata := &ContainerMetadata{}

	err = json.NewDecoder(bytes.NewReader(resp)).Decode(containerMetadata)

	if err != nil {
		return nil, fmt.Errorf("encountered unexpected error reading response from ECS Container Metadata Endpoint: %w", err)
	}

	return containerMetadata, nil
}
