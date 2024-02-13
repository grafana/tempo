// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter/internal/metadata"
)

var once sync.Once

// NewFactory creates a factory for Jaeger exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithTraces(createTracesExporter, metadata.TracesStability))
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutSettings: exporterhelper.NewDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:   exporterhelper.NewDefaultQueueSettings(),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
}

func logDeprecation(logger *zap.Logger) {
	once.Do(func() {
		logger.Warn("jaeger exporter is deprecated and will be removed in July 2023. See https://github.com/open-telemetry/opentelemetry-specification/pull/2858 for more details.")
	})
}

func createTracesExporter(
	_ context.Context,
	set exporter.CreateSettings,
	config component.Config,
) (exporter.Traces, error) {
	logDeprecation(set.Logger)
	expCfg := config.(*Config)
	return newTracesExporter(expCfg, set)
}
