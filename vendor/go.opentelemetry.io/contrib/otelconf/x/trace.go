// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

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
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/contrib/otelconf/internal/tls"
)

var errInvalidSamplerConfiguration = newErrInvalid("sampler configuration")

func tracerProvider(cfg configOptions, res *resource.Resource) (trace.TracerProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.TracerProvider == nil {
		return noop.NewTracerProvider(), noopShutdown, nil
	}

	opts := append(cfg.tracerProviderOptions, sdktrace.WithResource(res))

	var errs []error
	for _, processor := range cfg.opentelemetryConfig.TracerProvider.Processors {
		sp, err := spanProcessor(cfg.ctx, processor)
		if err == nil {
			opts = append(opts, sdktrace.WithSpanProcessor(sp))
		} else {
			errs = append(errs, err)
		}
	}
	if s, err := sampler(cfg.opentelemetryConfig.TracerProvider.Sampler); err == nil {
		opts = append(opts, sdktrace.WithSampler(s))
	} else {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return noop.NewTracerProvider(), noopShutdown, errors.Join(errs...)
	}
	tp := sdktrace.NewTracerProvider(opts...)
	return tp, tp.Shutdown, nil
}

func parentBasedSampler(s *ParentBasedSampler) (sdktrace.Sampler, error) {
	var rootSampler sdktrace.Sampler
	var opts []sdktrace.ParentBasedSamplerOption
	var errs []error
	var err error

	if s.Root == nil {
		rootSampler = sdktrace.AlwaysSample()
	} else {
		rootSampler, err = sampler(s.Root)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if s.RemoteParentSampled != nil {
		remoteParentSampler, err := sampler(s.RemoteParentSampled)
		if err != nil {
			errs = append(errs, err)
		} else {
			opts = append(opts, sdktrace.WithRemoteParentSampled(remoteParentSampler))
		}
	}
	if s.RemoteParentNotSampled != nil {
		remoteParentNotSampler, err := sampler(s.RemoteParentNotSampled)
		if err != nil {
			errs = append(errs, err)
		} else {
			opts = append(opts, sdktrace.WithRemoteParentNotSampled(remoteParentNotSampler))
		}
	}
	if s.LocalParentSampled != nil {
		localParentSampler, err := sampler(s.LocalParentSampled)
		if err != nil {
			errs = append(errs, err)
		} else {
			opts = append(opts, sdktrace.WithLocalParentSampled(localParentSampler))
		}
	}
	if s.LocalParentNotSampled != nil {
		localParentNotSampler, err := sampler(s.LocalParentNotSampled)
		if err != nil {
			errs = append(errs, err)
		} else {
			opts = append(opts, sdktrace.WithLocalParentNotSampled(localParentNotSampler))
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return sdktrace.ParentBased(rootSampler, opts...), nil
}

func sampler(s *Sampler) (sdktrace.Sampler, error) {
	if s == nil {
		// If omitted, parent based sampler with a root of always_on is used.
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	}
	if s.ParentBased != nil {
		return parentBasedSampler(s.ParentBased)
	}
	if s.AlwaysOff != nil {
		return sdktrace.NeverSample(), nil
	}
	if s.AlwaysOn != nil {
		return sdktrace.AlwaysSample(), nil
	}
	if s.TraceIDRatioBased != nil {
		if s.TraceIDRatioBased.Ratio == nil {
			return sdktrace.TraceIDRatioBased(1), nil
		}
		return sdktrace.TraceIDRatioBased(*s.TraceIDRatioBased.Ratio), nil
	}
	return nil, errInvalidSamplerConfiguration
}

func spanExporter(ctx context.Context, exporter SpanExporter) (sdktrace.SpanExporter, error) {
	exportersConfigured := 0
	var exportFunc func() (sdktrace.SpanExporter, error)

	if exporter.Console != nil {
		exportersConfigured++
		exportFunc = func() (sdktrace.SpanExporter, error) {
			return stdouttrace.New(
				stdouttrace.WithPrettyPrint(),
			)
		}
	}
	if exporter.OTLPHttp != nil {
		exportersConfigured++
		exportFunc = func() (sdktrace.SpanExporter, error) {
			return otlpHTTPSpanExporter(ctx, exporter.OTLPHttp)
		}
	}
	if exporter.OTLPGrpc != nil {
		exportersConfigured++
		exportFunc = func() (sdktrace.SpanExporter, error) {
			return otlpGRPCSpanExporter(ctx, exporter.OTLPGrpc)
		}
	}
	if exporter.OTLPFileDevelopment != nil {
		// TODO: implement file exporter https://github.com/open-telemetry/opentelemetry-go/issues/5408
		return nil, newErrInvalid("otlp_file/development")
	}

	if exportersConfigured > 1 {
		return nil, newErrInvalid("must not specify multiple exporters")
	}

	if exportFunc != nil {
		return exportFunc()
	}
	return nil, newErrInvalid("no valid span exporter")
}

func spanProcessor(ctx context.Context, processor SpanProcessor) (sdktrace.SpanProcessor, error) {
	if processor.Batch != nil && processor.Simple != nil {
		return nil, newErrInvalid("must not specify multiple span processor type")
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
	return nil, newErrInvalid("unsupported span processor type, must be one of simple or batch")
}

func otlpGRPCSpanExporter(ctx context.Context, otlpConfig *OTLPGrpcExporter) (sdktrace.SpanExporter, error) {
	var opts []otlptracegrpc.Option

	if otlpConfig.Endpoint != nil {
		u, err := url.ParseRequestURI(*otlpConfig.Endpoint)
		if err != nil {
			return nil, errors.Join(newErrInvalid("endpoint parsing failed"), err)
		}
		// ParseRequestURI leaves the Host field empty when no
		// scheme is specified (i.e. localhost:4317). This check is
		// here to support the case where a user may not specify a
		// scheme. The code does its best effort here by using
		// otlpConfig.Endpoint as-is in that case.
		if u.Host != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(u.Host))
		} else {
			opts = append(opts, otlptracegrpc.WithEndpoint(*otlpConfig.Endpoint))
		}

		if u.Scheme == "http" || (u.Scheme != "https" && otlpConfig.Tls != nil && otlpConfig.Tls.Insecure != nil && *otlpConfig.Tls.Insecure) {
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
			return nil, newErrInvalid(fmt.Sprintf("unsupported compression %q", *otlpConfig.Compression))
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracegrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	headersConfig, err := createHeadersConfig(otlpConfig.Headers, otlpConfig.HeadersList)
	if err != nil {
		return nil, err
	}
	if len(headersConfig) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(headersConfig))
	}

	if otlpConfig.Tls != nil && (otlpConfig.Tls.CaFile != nil || otlpConfig.Tls.CertFile != nil || otlpConfig.Tls.KeyFile != nil) {
		tlsConfig, err := tls.CreateConfig(otlpConfig.Tls.CaFile, otlpConfig.Tls.CertFile, otlpConfig.Tls.KeyFile)
		if err != nil {
			return nil, errors.Join(newErrInvalid("tls configuration"), err)
		}
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlptracegrpc.New(ctx, opts...)
}

func otlpHTTPSpanExporter(ctx context.Context, otlpConfig *OTLPHttpExporter) (sdktrace.SpanExporter, error) {
	var opts []otlptracehttp.Option

	if err := validateOTLPHTTPEncoding(otlpConfig.Encoding); err != nil {
		return nil, err
	}
	if otlpConfig.Endpoint != nil {
		u, err := url.ParseRequestURI(*otlpConfig.Endpoint)
		if err != nil {
			return nil, errors.Join(newErrInvalid("endpoint parsing failed"), err)
		}
		opts = append(opts, otlptracehttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if u.Path != "" {
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
			return nil, newErrInvalid(fmt.Sprintf("unsupported compression %q", *otlpConfig.Compression))
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracehttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	headersConfig, err := createHeadersConfig(otlpConfig.Headers, otlpConfig.HeadersList)
	if err != nil {
		return nil, err
	}
	if len(headersConfig) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(headersConfig))
	}

	if otlpConfig.Tls != nil {
		tlsConfig, err := tls.CreateConfig(otlpConfig.Tls.CaFile, otlpConfig.Tls.CertFile, otlpConfig.Tls.KeyFile)
		if err != nil {
			return nil, errors.Join(newErrInvalid("tls configuration"), err)
		}
		opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfig))
	}

	return otlptracehttp.New(ctx, opts...)
}

func batchSpanProcessor(bsp *BatchSpanProcessor, exp sdktrace.SpanExporter) (sdktrace.SpanProcessor, error) {
	var opts []sdktrace.BatchSpanProcessorOption
	if err := validateBatchSpanProcessor(bsp); err != nil {
		return nil, err
	}
	if bsp.ExportTimeout != nil {
		opts = append(opts, sdktrace.WithExportTimeout(time.Millisecond*time.Duration(*bsp.ExportTimeout)))
	}
	if bsp.MaxExportBatchSize != nil {
		opts = append(opts, sdktrace.WithMaxExportBatchSize(*bsp.MaxExportBatchSize))
	}
	if bsp.MaxQueueSize != nil {
		opts = append(opts, sdktrace.WithMaxQueueSize(*bsp.MaxQueueSize))
	}
	if bsp.ScheduleDelay != nil {
		opts = append(opts, sdktrace.WithBatchTimeout(time.Millisecond*time.Duration(*bsp.ScheduleDelay)))
	}
	return sdktrace.NewBatchSpanProcessor(exp, opts...), nil
}
