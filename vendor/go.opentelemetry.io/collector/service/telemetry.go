// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"net"
	"net/http"
	"strconv"

	ocmetric "go.opencensus.io/metric"
	"go.opencensus.io/metric/metricproducer"
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"
)

type telemetryInitializer struct {
	ocRegistry *ocmetric.Registry
	mp         metric.MeterProvider
	servers    []*http.Server

	disableHighCardinality bool
	extendedConfig         bool
}

func newColTelemetry(disableHighCardinality bool, extendedConfig bool) *telemetryInitializer {
	return &telemetryInitializer{
		mp:                     noopmetric.NewMeterProvider(),
		disableHighCardinality: disableHighCardinality,
		extendedConfig:         extendedConfig,
	}
}

func (tel *telemetryInitializer) init(res *resource.Resource, logger *zap.Logger, cfg telemetry.Config, asyncErrorChannel chan error) error {
	if cfg.Metrics.Level == configtelemetry.LevelNone || (cfg.Metrics.Address == "" && len(cfg.Metrics.Readers) == 0) {
		logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, cfg.Metrics.Address),
			zap.String(zapKeyTelemetryLevel, cfg.Metrics.Level.String()),
		)
		return nil
	}

	logger.Info("Setting up own telemetry...")
	return tel.initMetrics(res, logger, cfg, asyncErrorChannel)
}

func (tel *telemetryInitializer) initMetrics(res *resource.Resource, logger *zap.Logger, cfg telemetry.Config, asyncErrorChannel chan error) error {
	// Initialize the ocRegistry, still used by the process metrics.
	tel.ocRegistry = ocmetric.NewRegistry()

	if len(cfg.Metrics.Address) != 0 {
		if tel.extendedConfig {
			logger.Warn("service::telemetry::metrics::address is being deprecated in favor of service::telemetry::metrics::readers")
		}
		host, port, err := net.SplitHostPort(cfg.Metrics.Address)
		if err != nil {
			return err
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		if cfg.Metrics.Readers == nil {
			cfg.Metrics.Readers = []config.MetricReader{}
		}
		cfg.Metrics.Readers = append(cfg.Metrics.Readers, config.MetricReader{
			Pull: &config.PullMetricReader{
				Exporter: config.MetricExporter{
					Prometheus: &config.Prometheus{
						Host: &host,
						Port: &portInt,
					},
				},
			},
		})
	}

	metricproducer.GlobalManager().AddProducer(tel.ocRegistry)
	opts := []sdkmetric.Option{}
	for _, reader := range cfg.Metrics.Readers {
		// https://github.com/open-telemetry/opentelemetry-collector/issues/8045
		r, server, err := proctelemetry.InitMetricReader(context.Background(), reader, asyncErrorChannel)
		if err != nil {
			return err
		}
		if server != nil {
			tel.servers = append(tel.servers, server)
			logger.Info(
				"Serving metrics",
				zap.String(zapKeyTelemetryAddress, server.Addr),
				zap.String(zapKeyTelemetryLevel, cfg.Metrics.Level.String()),
			)
		}
		opts = append(opts, sdkmetric.WithReader(r))
	}

	mp, err := proctelemetry.InitOpenTelemetry(res, opts, tel.disableHighCardinality)
	if err != nil {
		return err
	}
	tel.mp = mp
	return nil
}

func (tel *telemetryInitializer) shutdown() error {
	metricproducer.GlobalManager().DeleteProducer(tel.ocRegistry)

	var errs error
	for _, server := range tel.servers {
		if server != nil {
			errs = multierr.Append(errs, server.Close())
		}
	}
	return errs
}
