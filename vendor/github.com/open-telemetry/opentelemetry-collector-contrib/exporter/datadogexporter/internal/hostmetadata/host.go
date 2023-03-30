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

package hostmetadata // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata"

import (
	"context"
	"fmt"

	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	gocache "github.com/patrickmn/go-cache"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/azure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/ec2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/ecs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/gcp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/k8s"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/internal/system"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/provider"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/valid"
)

var HostnamePreviewFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"exporter.datadog.hostname.preview",
	featuregate.StageBeta,
	featuregate.WithRegisterDescription("Use the 'preview' hostname resolution rules, which are consistent with Datadog cloud integration hostname resolution rules, and set 'host_metadata::hostname_source' to 'config_or_system' by default."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/10424"),
)

func buildPreviewProvider(set component.TelemetrySettings, configHostname string) (source.Provider, error) {
	ecs, err := ecs.NewProvider(set)
	if err != nil {
		return nil, fmt.Errorf("failed to build ECS Fargate provider: %w", err)
	}

	azureProvider := azure.NewProvider()
	ec2Provider, err := ec2.NewProvider(set.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build EC2 provider: %w", err)
	}
	gcpProvider := gcp.NewProvider()

	clusterNameProvider, err := provider.ChainCluster(set.Logger,
		map[string]provider.ClusterNameProvider{
			"azure": azureProvider,
			"ec2":   ec2Provider,
			"gcp":   gcpProvider,
		},
		[]string{"azure", "ec2", "gcp"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build Kubernetes cluster name provider: %w", err)
	}

	k8sProvider, err := k8s.NewProvider(set.Logger, clusterNameProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to build Kubernetes hostname provider: %w", err)
	}

	chain, err := provider.Chain(
		set.Logger,
		map[string]source.Provider{
			"config":     provider.Config(configHostname),
			"azure":      azureProvider,
			"ecs":        ecs,
			"ec2":        ec2Provider,
			"gcp":        gcpProvider,
			"kubernetes": k8sProvider,
			"system":     system.NewProvider(set.Logger),
		},
		[]string{"config", "azure", "ecs", "ec2", "gcp", "kubernetes", "system"},
	)

	if err != nil {
		return nil, err
	}

	return provider.Once(chain), nil
}

func buildCurrentProvider(set component.TelemetrySettings, configHostname string) (source.Provider, error) {
	ec2Provider, err := ec2.NewProvider(set.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build EC2 provider: %w", err)
	}

	return &currentProvider{
		logger:         set.Logger,
		configHostname: configHostname,
		systemProvider: system.NewProvider(set.Logger),
		ec2Provider:    ec2Provider,
	}, nil
}

func GetSourceProvider(set component.TelemetrySettings, configHostname string) (source.Provider, error) {
	if HostnamePreviewFeatureGate.IsEnabled() {
		return buildPreviewProvider(set, configHostname)
	}

	return buildCurrentProvider(set, configHostname)
}

var _ source.Provider = (*currentProvider)(nil)

type currentProvider struct {
	logger         *zap.Logger
	configHostname string
	systemProvider *system.Provider
	ec2Provider    *ec2.Provider
}

// Hostname gets the hostname according to configuration.
// It checks in the following order
// 1. Configuration
// 2. Cache
// 3. EC2 instance metadata
// 4. System
func (c *currentProvider) hostname(ctx context.Context) string {
	if c.configHostname != "" {
		return c.configHostname
	}

	if cacheVal, ok := hostnameCache.Get(cacheKeyHostname); ok {
		return cacheVal.(string)
	}

	ec2Info := c.ec2Provider.HostInfo()
	hostname := ec2Info.GetHostname(c.logger)

	if hostname == "" {
		// Get system hostname
		var err error
		src, err := c.systemProvider.Source(ctx)
		if err != nil {
			c.logger.Debug("system provider is unavailable", zap.Error(err))
		} else {
			hostname = src.Identifier
		}
	}

	if err := valid.Hostname(hostname); err != nil {
		// If invalid log but continue
		c.logger.Error("Detected hostname is not valid", zap.Error(err))
	}

	c.logger.Debug("Canonical hostname automatically set", zap.String("hostname", hostname))
	hostnameCache.Set(cacheKeyHostname, hostname, gocache.NoExpiration)
	return hostname
}

func (c *currentProvider) Source(ctx context.Context) (source.Source, error) {
	return source.Source{Kind: source.HostnameKind, Identifier: c.hostname(ctx)}, nil
}
