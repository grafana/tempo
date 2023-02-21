// Copyright The OpenTelemetry Authors
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

// This file contains code based on the Azure IMDS samples, https://github.com/microsoft/azureimds
// under the Apache License 2.0

package azure // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/metadataproviders/azure"

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// Azure IMDS compute endpoint, see https://aka.ms/azureimds
	metadataEndpoint = "http://169.254.169.254/metadata/instance/compute"
)

// Provider gets metadata from the Azure IMDS.
type Provider interface {
	Metadata(context.Context) (*ComputeMetadata, error)
}

type azureProviderImpl struct {
	endpoint string
	client   *http.Client
}

// NewProvider creates a new metadata provider
func NewProvider() Provider {
	return &azureProviderImpl{
		endpoint: metadataEndpoint,
		client:   &http.Client{},
	}
}

// ComputeMetadata is the Azure IMDS compute metadata response format
type ComputeMetadata struct {
	Location          string `json:"location"`
	Name              string `json:"name"`
	VMID              string `json:"vmID"`
	VMSize            string `json:"vmSize"`
	SubscriptionID    string `json:"subscriptionID"`
	ResourceGroupName string `json:"resourceGroupName"`
	VMScaleSetName    string `json:"vmScaleSetName"`
}

// Metadata queries a given endpoint and parses the output to the Azure IMDS format
func (p *azureProviderImpl) Metadata(ctx context.Context) (*ComputeMetadata, error) {
	const (
		// API version used
		apiVersionKey = "api-version"
		apiVersion    = "2020-09-01"

		// format used
		formatKey  = "format"
		jsonFormat = "json"
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Metadata", "True")
	q := req.URL.Query()
	q.Add(formatKey, jsonFormat)
	q.Add(apiVersionKey, apiVersion)
	req.URL.RawQuery = q.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query Azure IMDS: %w", err)
	} else if resp.StatusCode != 200 {
		//lint:ignore ST1005 Azure is a capitalized proper noun here
		return nil, fmt.Errorf("Azure IMDS replied with status code: %s", resp.Status)
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Azure IMDS reply: %w", err)
	}

	var metadata *ComputeMetadata
	err = json.Unmarshal(respBody, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Azure IMDS reply: %w", err)
	}

	return metadata, nil
}
