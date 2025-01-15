// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter/internal/metadata"
)

const (
	defaultTimeout = time.Second * 5

	defaultFormat = "json"

	defaultServiceName string = "<missing service name>"
)

// NewFactory creates a factory for Zipkin exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithTraces(createTracesExporter, metadata.TracesStability))
}

func createDefaultConfig() component.Config {
	defaultClientHTTPSettings := confighttp.NewDefaultClientConfig()
	defaultClientHTTPSettings.Timeout = defaultTimeout
	defaultClientHTTPSettings.WriteBufferSize = 512 * 1024
	return &Config{
		BackOffConfig:      configretry.NewDefaultBackOffConfig(),
		QueueSettings:      exporterhelper.NewDefaultQueueConfig(),
		ClientConfig:       defaultClientHTTPSettings,
		Format:             defaultFormat,
		DefaultServiceName: defaultServiceName,
	}
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	zc := cfg.(*Config)

	ze, err := createZipkinExporter(zc, set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		ze.pushTraces,
		exporterhelper.WithStart(ze.start),
		// explicitly disable since we rely on http.Client timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithQueue(zc.QueueSettings),
		exporterhelper.WithRetry(zc.BackOffConfig))
}
