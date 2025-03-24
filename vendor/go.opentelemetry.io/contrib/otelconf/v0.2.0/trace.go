// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.2.0"

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func tracerProvider(cfg configOptions, res *resource.Resource) (trace.TracerProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.TracerProvider == nil {
		return noop.NewTracerProvider(), noopShutdown, nil
	}
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}
	var errs []error
	for _, processor := range cfg.opentelemetryConfig.TracerProvider.Processors {
		sp, err := spanProcessor(cfg.ctx, processor)
		if err == nil {
			opts = append(opts, sdktrace.WithSpanProcessor(sp))
		} else {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return noop.NewTracerProvider(), noopShutdown, errors.Join(errs...)
	}
	tp := sdktrace.NewTracerProvider(opts...)
	return tp, tp.Shutdown, nil
}

func spanExporter(ctx context.Context, exporter SpanExporter) (sdktrace.SpanExporter, error) {
	if exporter.Console != nil && exporter.OTLP != nil {
		return nil, errors.New("must not specify multiple exporters")
	}

	if exporter.Console != nil {
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
	}
	if exporter.OTLP != nil {
		switch exporter.OTLP.Protocol {
		case protocolProtobufHTTP:
			return otlpHTTPSpanExporter(ctx, exporter.OTLP)
		case protocolProtobufGRPC:
			return otlpGRPCSpanExporter(ctx, exporter.OTLP)
		default:
			return nil, fmt.Errorf("unsupported protocol %q", exporter.OTLP.Protocol)
		}
	}
	return nil, errors.New("no valid span exporter")
}

func spanProcessor(ctx context.Context, processor SpanProcessor) (sdktrace.SpanProcessor, error) {
	if processor.Batch != nil && processor.Simple != nil {
		return nil, errors.New("must not specify multiple span processor type")
	}
	if processor.Batch != nil {
		exp, err := spanExporter(ctx, processor.Batch.Exporter)
		if err != nil {
			return nil, err
		}
		return batchSpanProcessor(processor.Batch, exp)
	}
	if processor.Simple != nil {
		exp, err := spanExporter(ctx, processor.Simple.Exporter)
		if err != nil {
			return nil, err
		}
		return sdktrace.NewSimpleSpanProcessor(exp), nil
	}
	return nil, errors.New("unsupported span processor type, must be one of simple or batch")
}

func otlpGRPCSpanExporter(ctx context.Context, otlpConfig *OTLP) (sdktrace.SpanExporter, error) {
	var opts []otlptracegrpc.Option

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(otlpConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		// ParseRequestURI leaves the Host field empty when no
		// scheme is specified (i.e. localhost:4317). This check is
		// here to support the case where a user may not specify a
		// scheme. The code does its best effort here by using
		// otlpConfig.Endpoint as-is in that case.
		if u.Host != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(u.Host))
		} else {
			opts = append(opts, otlptracegrpc.WithEndpoint(otlpConfig.Endpoint))
		}

		if u.Scheme == "http" {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlptracegrpc.WithCompressor(*otlpConfig.Compression))
		case compressionNone:
			// none requires no options
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracegrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(otlpConfig.Headers))
	}

	return otlptracegrpc.New(ctx, opts...)
}

func otlpHTTPSpanExporter(ctx context.Context, otlpConfig *OTLP) (sdktrace.SpanExporter, error) {
	var opts []otlptracehttp.Option

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(otlpConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracehttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(u.Path) > 0 {
			opts = append(opts, otlptracehttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		case compressionNone:
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracehttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(otlpConfig.Headers))
	}

	return otlptracehttp.New(ctx, opts...)
}

func batchSpanProcessor(bsp *BatchSpanProcessor, exp sdktrace.SpanExporter) (sdktrace.SpanProcessor, error) {
	var opts []sdktrace.BatchSpanProcessorOption
	if bsp.ExportTimeout != nil {
		if *bsp.ExportTimeout < 0 {
			return nil, fmt.Errorf("invalid export timeout %d", *bsp.ExportTimeout)
		}
		opts = append(opts, sdktrace.WithExportTimeout(time.Millisecond*time.Duration(*bsp.ExportTimeout)))
	}
	if bsp.MaxExportBatchSize != nil {
		if *bsp.MaxExportBatchSize < 0 {
			return nil, fmt.Errorf("invalid batch size %d", *bsp.MaxExportBatchSize)
		}
		opts = append(opts, sdktrace.WithMaxExportBatchSize(*bsp.MaxExportBatchSize))
	}
	if bsp.MaxQueueSize != nil {
		if *bsp.MaxQueueSize < 0 {
			return nil, fmt.Errorf("invalid queue size %d", *bsp.MaxQueueSize)
		}
		opts = append(opts, sdktrace.WithMaxQueueSize(*bsp.MaxQueueSize))
	}
	if bsp.ScheduleDelay != nil {
		if *bsp.ScheduleDelay < 0 {
			return nil, fmt.Errorf("invalid schedule delay %d", *bsp.ScheduleDelay)
		}
		opts = append(opts, sdktrace.WithBatchTimeout(time.Millisecond*time.Duration(*bsp.ScheduleDelay)))
	}
	return sdktrace.NewBatchSpanProcessor(exp, opts...), nil
}
