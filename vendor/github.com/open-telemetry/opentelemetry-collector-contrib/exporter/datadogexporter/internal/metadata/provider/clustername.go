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

package provider // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metadata/provider"

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type ClusterNameProvider interface {
	ClusterName(context.Context) (string, error)
}

var _ ClusterNameProvider = (*chainClusterProvider)(nil)

type chainClusterProvider struct {
	logger       *zap.Logger
	providers    map[string]ClusterNameProvider
	priorityList []string
}

func (p *chainClusterProvider) ClusterName(ctx context.Context) (string, error) {
	for _, source := range p.priorityList {
		zapSource := zap.String("source", source)
		provider := p.providers[source]
		clusterName, err := provider.ClusterName(ctx)
		if err == nil {
			p.logger.Info("Resolved cluster name", zapSource, zap.String("cluster name", clusterName))
			return clusterName, nil
		}
		p.logger.Debug("Unavailable cluster name provider", zapSource, zap.Error(err))
	}

	return "", fmt.Errorf("no cluster name provider was available")
}

// Chain providers into a single provider that returns the first available hostname.
func ChainCluster(logger *zap.Logger, providers map[string]ClusterNameProvider, priorityList []string) (ClusterNameProvider, error) {
	for _, source := range priorityList {
		if _, ok := providers[source]; !ok {
			return nil, fmt.Errorf("%q source is not available in providers", source)
		}
	}

	return &chainClusterProvider{logger: logger, providers: providers, priorityList: priorityList}, nil
}
