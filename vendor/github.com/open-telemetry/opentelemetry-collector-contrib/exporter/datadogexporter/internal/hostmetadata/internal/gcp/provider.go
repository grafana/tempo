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

// Package gcp contains the GCP hostname provider
package gcp // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/gcp"

import (
	"context"
	"fmt"
	"strings"

	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/provider"
)

var _ source.Provider = (*Provider)(nil)
var _ provider.ClusterNameProvider = (*Provider)(nil)

var _ gcpDetector = gcp.NewDetector()

type gcpDetector interface {
	ProjectID() (string, error)
	CloudPlatform() gcp.Platform
	GCEHostName() (string, error)
	GKEClusterName() (string, error)
}

type Provider struct {
	detector gcpDetector
}

func platformDescription(platform gcp.Platform) string {
	switch platform {
	case gcp.UnknownPlatform:
		return "Unknown platform"
	case gcp.GKE:
		return "Google Kubernetes Engine"
	case gcp.GCE:
		return "Google Cloud Engine"
	case gcp.CloudRun:
		return "Google Cloud Run"
	case gcp.CloudFunctions:
		return "Google Cloud Functions"
	case gcp.AppEngineStandard, gcp.AppEngineFlex:
		return "Google AppEngine"
	}
	return "Unrecognized platform"
}

// Hostname returns the GCP cloud integration hostname.
func (p *Provider) Source(context.Context) (source.Source, error) {
	if platform := p.detector.CloudPlatform(); platform != gcp.GCE && platform != gcp.GKE {
		return source.Source{}, fmt.Errorf("not on GCE or GKE (platform: %s)", platformDescription(platform))
	}

	name, err := p.detector.GCEHostName()
	if err != nil {
		return source.Source{}, fmt.Errorf("failed to get instance name: %w", err)
	}

	// Use the same logic as in the metadata from attributes logic.
	if strings.Count(name, ".") >= 3 {
		name = strings.SplitN(name, ".", 2)[0]
	}

	cloudAccount, err := p.detector.ProjectID()
	if err != nil {
		return source.Source{}, fmt.Errorf("failed to get project ID: %w", err)
	}

	return source.Source{Kind: source.HostnameKind, Identifier: fmt.Sprintf("%s.%s", name, cloudAccount)}, nil
}

func (p *Provider) ClusterName(ctx context.Context) (string, error) {
	return p.detector.GKEClusterName()
}

// NewProvider creates a new GCP hostname provider.
func NewProvider() *Provider {
	return &Provider{detector: gcp.NewDetector()}
}
