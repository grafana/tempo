// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
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
	return &Config{
		RetrySettings: exporterhelper.NewDefaultRetrySettings(),
		QueueSettings: exporterhelper.NewDefaultQueueSettings(),
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Timeout: defaultTimeout,
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
		Format:             defaultFormat,
		DefaultServiceName: defaultServiceName,
	}
}

func createTracesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Traces, error) {
	zc := cfg.(*Config)

	if zc.Endpoint == "" {
		// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/215
		return nil, errors.New("exporter config requires a non-empty 'endpoint'")
	}

	ze, err := createZipkinExporter(zc, set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		ze.pushTraces,
		exporterhelper.WithStart(ze.start),
		// explicitly disable since we rely on http.Client timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithQueue(zc.QueueSettings),
		exporterhelper.WithRetry(zc.RetrySettings))
}
