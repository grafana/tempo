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

// Package provider contains the cluster name provider
package provider // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/provider"

import (
	"context"
	"fmt"
	"sync"

	"github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes/source"
	"go.uber.org/zap"
)

var _ source.Provider = (*chainProvider)(nil)

type chainProvider struct {
	logger       *zap.Logger
	providers    map[string]source.Provider
	priorityList []string
}

func (p *chainProvider) Source(ctx context.Context) (source.Source, error) {
	for _, source := range p.priorityList {
		zapProvider := zap.String("provider", source)
		provider := p.providers[source]
		src, err := provider.Source(ctx)
		if err == nil {
			p.logger.Info("Resolved source", zapProvider, zap.Any("source", src))
			return src, nil
		}
		p.logger.Debug("Unavailable source provider", zapProvider, zap.Error(err))
	}

	return source.Source{}, fmt.Errorf("no source provider was available")
}

// Chain providers into a single provider that returns the first available hostname.
func Chain(logger *zap.Logger, providers map[string]source.Provider, priorityList []string) (source.Provider, error) {
	for _, source := range priorityList {
		if _, ok := providers[source]; !ok {
			return nil, fmt.Errorf("%q source is not available in providers", source)
		}
	}

	return &chainProvider{logger: logger, providers: providers, priorityList: priorityList}, nil
}

var _ source.Provider = (*configProvider)(nil)

type configProvider struct {
	hostname string
}

func (p *configProvider) Source(context.Context) (source.Source, error) {
	if p.hostname == "" {
		return source.Source{}, fmt.Errorf("empty configuration hostname")
	}
	return source.Source{Kind: source.HostnameKind, Identifier: p.hostname}, nil
}

// Config returns fixed hostname.
func Config(hostname string) source.Provider {
	return &configProvider{hostname}
}

var _ source.Provider = (*onceProvider)(nil)

type onceProvider struct {
	once     sync.Once
	src      source.Source
	err      error
	provider source.Provider
}

func (c *onceProvider) Source(ctx context.Context) (source.Source, error) {
	c.once.Do(func() {
		c.src, c.err = c.provider.Source(ctx)
	})

	return c.src, c.err
}

// Once wraps a provider to call it only once.
func Once(provider source.Provider) source.Provider {
	return &onceProvider{
		provider: provider,
	}
}
