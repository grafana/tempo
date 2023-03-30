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

// Package ecs contains the ECS Fargate hostname provider
package ecs // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/ecs"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil/endpoints"
)

var ErrNotOnECSFargate = fmt.Errorf("not running on ECS Fargate")

var _ source.Provider = (*Provider)(nil)

type Provider struct {
	missingEndpoint bool
	ecsMetadata     ecsutil.MetadataProvider
}

// OnECSFargate determines if the application is running on ECS Fargate.
func (p *Provider) OnECSFargate(ctx context.Context) (bool, error) {
	if p.missingEndpoint {
		// No ECS metadata endpoint, therefore not on ECS Fargate
		return false, nil
	}

	tmdeResp, err := p.ecsMetadata.FetchTaskMetadata()
	if err != nil {
		return false, fmt.Errorf("failed to fetch task metadata: %w", err)
	}

	switch lt := strings.ToLower(tmdeResp.LaunchType); lt {
	case "ec2":
		return false, nil
	case "fargate":
		return true, nil
	}

	return false, fmt.Errorf("TMDE endpoint is queryable, but launch type is unavailable")
}

// Source returns the task ARN of the ECS Fargate task if on ECS Fargate.
func (p *Provider) Source(ctx context.Context) (source.Source, error) {
	if onECSFargate, err := p.OnECSFargate(ctx); !onECSFargate && err == nil {
		// Not on ECS Fargate
		return source.Source{}, ErrNotOnECSFargate
	} else if err != nil {
		// Failed to determine if on ECS Fargate
		return source.Source{}, err
	}

	tmdeResp, err := p.ecsMetadata.FetchTaskMetadata()
	if err != nil {
		return source.Source{}, err
	}

	return source.Source{
		Kind:       source.AWSECSFargateKind,
		Identifier: tmdeResp.TaskARN,
	}, nil
}

// NewProvider creates a new ECS Fargate hostname provider.
func NewProvider(set component.TelemetrySettings) (*Provider, error) {
	ecsMetadata, err := ecsutil.NewDetectedTaskMetadataProvider(set)
	if err != nil {
		// Metadata endpoint has not been detected
		var errNotDetected endpoints.ErrNoTaskMetadataEndpointDetected
		if ok := errors.As(err, &errNotDetected); ok {
			return &Provider{missingEndpoint: true, ecsMetadata: nil}, nil
		}
		return nil, fmt.Errorf("unable to create task metadata provider: %w", err)
	}
	return &Provider{missingEndpoint: false, ecsMetadata: ecsMetadata}, nil
}
