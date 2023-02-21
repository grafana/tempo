// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ecsutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil"

import (
	"net/url"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

func NewRestClient(baseEndpoint url.URL, clientSettings confighttp.HTTPClientSettings, settings component.TelemetrySettings) (RestClient, error) {
	clientProvider := NewClientProvider(baseEndpoint, clientSettings, &nopHost{}, settings)

	client, err := clientProvider.BuildClient()
	if err != nil {
		return nil, err
	}
	return NewRestClientFromClient(client), nil
}

// TODO: Instead of using this, expose it as a argument to NewRestClient.
type nopHost struct {
	component.Host
}

func (nh *nopHost) GetExtensions() map[component.ID]component.Component {
	return map[component.ID]component.Component{}
}

// RestClient is swappable for testing.
type RestClient interface {
	GetResponse(path string) ([]byte, error)
}

// TaskMetadataRestClient is a thin wrapper around an ecs task metadata client, encapsulating endpoints
// and their corresponding http methods.
type TaskMetadataRestClient struct {
	client Client
}

// NewRestClientFromClient creates a new copy of the Client
func NewRestClientFromClient(client Client) *TaskMetadataRestClient {
	return &TaskMetadataRestClient{client: client}
}

// GetResponse gets the desired path from the configured metadata endpoint
func (c *TaskMetadataRestClient) GetResponse(path string) ([]byte, error) {
	response, err := c.client.Get(path)
	if err != nil {
		return nil, err
	}
	return response, nil
}
