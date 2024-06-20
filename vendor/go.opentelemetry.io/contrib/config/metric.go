// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func meterProvider(cfg configOptions, res *resource.Resource) (metric.MeterProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.MeterProvider == nil {
		return noop.NewMeterProvider(), noopShutdown, nil
	}
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}

	var errs []error
	for _, reader := range cfg.opentelemetryConfig.MeterProvider.Readers {
		r, err := metricReader(cfg.ctx, reader)
		if err == nil {
			opts = append(opts, sdkmetric.WithReader(r))
		} else {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return noop.NewMeterProvider(), noopShutdown, errors.Join(errs...)
	}

	mp := sdkmetric.NewMeterProvider(opts...)
	return mp, mp.Shutdown, nil
}

func metricReader(ctx context.Context, r MetricReader) (sdkmetric.Reader, error) {
	if r.Periodic != nil && r.Pull != nil {
		return nil, errors.New("must not specify multiple metric reader type")
	}

	if r.Periodic != nil {
		return periodicExporter(ctx, r.Periodic.Exporter)
	}

	if r.Pull != nil {
		return pullReader(ctx, r.Pull.Exporter)
	}
	return nil, errors.New("no valid metric reader")
}

func pullReader(ctx context.Context, exporter MetricExporter) (sdkmetric.Reader, error) {
	if exporter.Prometheus != nil {
		return prometheusReader(ctx, exporter.Prometheus)
	}
	return nil, errors.New("no valid metric exporter")
}

func periodicExporter(ctx context.Context, exporter MetricExporter, opts ...sdkmetric.PeriodicReaderOption) (sdkmetric.Reader, error) {
	if exporter.Console != nil && exporter.OTLP != nil {
		return nil, errors.New("must not specify multiple exporters")
	}
	if exporter.Console != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		exp, err := stdoutmetric.New(
			stdoutmetric.WithEncoder(enc),
		)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil
	}
	if exporter.OTLP != nil {
		var err error
		var exp sdkmetric.Exporter
		switch exporter.OTLP.Protocol {
		case protocolProtobufHTTP:
			exp, err = otlpHTTPMetricExporter(ctx, exporter.OTLP)
		case protocolProtobufGRPC:
			exp, err = otlpGRPCMetricExporter(ctx, exporter.OTLP)
		default:
			return nil, fmt.Errorf("unsupported protocol %q", exporter.OTLP.Protocol)
		}
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil
	}
	return nil, errors.New("no valid metric exporter")
}

func otlpHTTPMetricExporter(ctx context.Context, otlpConfig *OTLPMetric) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(otlpConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(u.Path) > 0 {
			opts = append(opts, otlpmetrichttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
		case compressionNone:
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.NoCompression))
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil {
		opts = append(opts, otlpmetrichttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(otlpConfig.Headers))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

func otlpGRPCMetricExporter(ctx context.Context, otlpConfig *OTLPMetric) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(otlpConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		// ParseRequestURI leaves the Host field empty when no
		// scheme is specified (i.e. localhost:4317). This check is
		// here to support the case where a user may not specify a
		// scheme. The code does its best effort here by using
		// otlpConfig.Endpoint as-is in that case
		if u.Host != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(u.Host))
		} else {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(otlpConfig.Endpoint))
		}
		if u.Scheme == "http" {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlpmetricgrpc.WithCompressor(*otlpConfig.Compression))
		case compressionNone:
			// none requires no options
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlpmetricgrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(otlpConfig.Headers))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

func prometheusReader(ctx context.Context, prometheusConfig *Prometheus) (sdkmetric.Reader, error) {
	var opts []otelprom.Option
	if prometheusConfig.Host == nil {
		return nil, fmt.Errorf("host must be specified")
	}
	if prometheusConfig.Port == nil {
		return nil, fmt.Errorf("port must be specified")
	}
	if prometheusConfig.WithoutScopeInfo != nil && *prometheusConfig.WithoutScopeInfo {
		opts = append(opts, otelprom.WithoutScopeInfo())
	}
	if prometheusConfig.WithoutTypeSuffix != nil && *prometheusConfig.WithoutTypeSuffix {
		opts = append(opts, otelprom.WithoutCounterSuffixes())
	}
	if prometheusConfig.WithoutUnits != nil && *prometheusConfig.WithoutUnits {
		opts = append(opts, otelprom.WithoutUnits())
	}

	reg := prometheus.NewRegistry()
	opts = append(opts, otelprom.WithRegisterer(reg))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	server := http.Server{
		// Timeouts are necessary to make a server resilent to attacks, but ListenAndServe doesn't set any.
		// We use values from this example: https://blog.cloudflare.com/exposing-go-on-the-internet/#:~:text=There%20are%20three%20main%20timeouts
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
	addr := fmt.Sprintf("%s:%d", *prometheusConfig.Host, *prometheusConfig.Port)

	// TODO: add support for constant label filter
	// 	otelprom.WithResourceAsConstantLabels(attribute.NewDenyKeysFilter()),
	// )
	reader, err := otelprom.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating otel prometheus exporter: %w", err)
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("binding address %s for Prometheus exporter: %w", addr, err),
			reader.Shutdown(ctx),
		)
	}

	go func() {
		if err := server.Serve(lis); err != nil && err != http.ErrServerClosed {
			otel.Handle(fmt.Errorf("the Prometheus HTTP server exited unexpectedly: %w", err))
		}
	}()

	return readerWithServer{reader, &server}, nil
}

type readerWithServer struct {
	sdkmetric.Reader
	server *http.Server
}

func (rws readerWithServer) Shutdown(ctx context.Context) error {
	return errors.Join(
		rws.Reader.Shutdown(ctx),
		rws.server.Shutdown(ctx),
	)
}
