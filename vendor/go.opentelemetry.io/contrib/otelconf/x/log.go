// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/contrib/otelconf/internal/tls"
)

func loggerProvider(cfg configOptions, res *resource.Resource) (log.LoggerProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.LoggerProvider == nil {
		return noop.NewLoggerProvider(), noopShutdown, nil
	}
	opts := append(cfg.loggerProviderOptions, sdklog.WithResource(res))

	var errs []error
	for _, processor := range cfg.opentelemetryConfig.LoggerProvider.Processors {
		sp, err := logProcessor(cfg.ctx, processor)
		if err == nil {
			opts = append(opts, sdklog.WithProcessor(sp))
		} else {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return noop.NewLoggerProvider(), noopShutdown, errors.Join(errs...)
	}

	lp := sdklog.NewLoggerProvider(opts...)
	return lp, lp.Shutdown, nil
}

func logProcessor(ctx context.Context, processor LogRecordProcessor) (sdklog.Processor, error) {
	if processor.Batch != nil && processor.Simple != nil {
		return nil, newErrInvalid("must not specify multiple log processor type")
	}
	if processor.Batch != nil {
		exp, err := logExporter(ctx, processor.Batch.Exporter)
		if err != nil {
			return nil, err
		}
		return batchLogProcessor(processor.Batch, exp)
	}
	if processor.Simple != nil {
		exp, err := logExporter(ctx, processor.Simple.Exporter)
		if err != nil {
			return nil, err
		}
		return sdklog.NewSimpleProcessor(exp), nil
	}
	return nil, newErrInvalid("unsupported log processor type, must be one of simple or batch")
}

func logExporter(ctx context.Context, exporter LogRecordExporter) (sdklog.Exporter, error) {
	exportersConfigured := 0
	var exportFunc func() (sdklog.Exporter, error)

	if exporter.Console != nil {
		exportersConfigured++
		exportFunc = func() (sdklog.Exporter, error) {
			return stdoutlog.New(
				stdoutlog.WithPrettyPrint(),
			)
		}
	}

	if exporter.OTLPHttp != nil {
		exportersConfigured++
		exportFunc = func() (sdklog.Exporter, error) {
			return otlpHTTPLogExporter(ctx, exporter.OTLPHttp)
		}
	}
	if exporter.OTLPGrpc != nil {
		exportersConfigured++
		exportFunc = func() (sdklog.Exporter, error) {
			return otlpGRPCLogExporter(ctx, exporter.OTLPGrpc)
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

	return nil, newErrInvalid("no valid log exporter")
}

func batchLogProcessor(blp *BatchLogRecordProcessor, exp sdklog.Exporter) (*sdklog.BatchProcessor, error) {
	var opts []sdklog.BatchProcessorOption
	if err := validateBatchLogRecordProcessor(blp); err != nil {
		return nil, err
	}
	if blp.ExportTimeout != nil {
		opts = append(opts, sdklog.WithExportTimeout(time.Millisecond*time.Duration(*blp.ExportTimeout)))
	}
	if blp.MaxExportBatchSize != nil {
		opts = append(opts, sdklog.WithExportMaxBatchSize(*blp.MaxExportBatchSize))
	}
	if blp.MaxQueueSize != nil {
		opts = append(opts, sdklog.WithMaxQueueSize(*blp.MaxQueueSize))
	}

	if blp.ScheduleDelay != nil {
		opts = append(opts, sdklog.WithExportInterval(time.Millisecond*time.Duration(*blp.ScheduleDelay)))
	}

	return sdklog.NewBatchProcessor(exp, opts...), nil
}

func otlpHTTPLogExporter(ctx context.Context, otlpConfig *OTLPHttpExporter) (sdklog.Exporter, error) {
	var opts []otlploghttp.Option

	if err := validateOTLPHTTPEncoding(otlpConfig.Encoding); err != nil {
		return nil, err
	}
	if otlpConfig.Endpoint != nil {
		u, err := url.ParseRequestURI(*otlpConfig.Endpoint)
		if err != nil {
			return nil, errors.Join(newErrInvalid("endpoint parsing failed"), err)
		}
		opts = append(opts, otlploghttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		if u.Path != "" {
			opts = append(opts, otlploghttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlploghttp.WithCompression(otlploghttp.GzipCompression))
		case compressionNone:
			opts = append(opts, otlploghttp.WithCompression(otlploghttp.NoCompression))
		default:
			return nil, newErrInvalid(fmt.Sprintf("unsupported compression %q", *otlpConfig.Compression))
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlploghttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	headersConfig, err := createHeadersConfig(otlpConfig.Headers, otlpConfig.HeadersList)
	if err != nil {
		return nil, err
	}
	if len(headersConfig) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(headersConfig))
	}

	if otlpConfig.Tls != nil {
		tlsConfig, err := tls.CreateConfig(otlpConfig.Tls.CaFile, otlpConfig.Tls.CertFile, otlpConfig.Tls.KeyFile)
		if err != nil {
			return nil, errors.Join(newErrInvalid("tls configuration"), err)
		}
		opts = append(opts, otlploghttp.WithTLSClientConfig(tlsConfig))
	}

	return otlploghttp.New(ctx, opts...)
}

func otlpGRPCLogExporter(ctx context.Context, otlpConfig *OTLPGrpcExporter) (sdklog.Exporter, error) {
	var opts []otlploggrpc.Option

	if otlpConfig.Endpoint != nil {
		u, err := url.ParseRequestURI(*otlpConfig.Endpoint)
		if err != nil {
			return nil, errors.Join(newErrInvalid("endpoint parsing failed"), err)
		}
		// ParseRequestURI leaves the Host field empty when no
		// scheme is specified (i.e. localhost:4317). This check is
		// here to support the case where a user may not specify a
		// scheme. The code does its best effort here by using
		// otlpConfig.Endpoint as-is in that case
		if u.Host != "" {
			opts = append(opts, otlploggrpc.WithEndpoint(u.Host))
		} else {
			opts = append(opts, otlploggrpc.WithEndpoint(*otlpConfig.Endpoint))
		}
		if u.Scheme == "http" || (u.Scheme != "https" && otlpConfig.Tls != nil && otlpConfig.Tls.Insecure != nil && *otlpConfig.Tls.Insecure) {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case compressionGzip:
			opts = append(opts, otlploggrpc.WithCompressor(*otlpConfig.Compression))
		case compressionNone:
			// none requires no options
		default:
			return nil, newErrInvalid(fmt.Sprintf("unsupported compression %q", *otlpConfig.Compression))
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlploggrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	headersConfig, err := createHeadersConfig(otlpConfig.Headers, otlpConfig.HeadersList)
	if err != nil {
		return nil, err
	}
	if len(headersConfig) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(headersConfig))
	}

	if otlpConfig.Tls != nil && (otlpConfig.Tls.CaFile != nil || otlpConfig.Tls.CertFile != nil || otlpConfig.Tls.KeyFile != nil) {
		tlsConfig, err := tls.CreateConfig(otlpConfig.Tls.CaFile, otlpConfig.Tls.CertFile, otlpConfig.Tls.KeyFile)
		if err != nil {
			return nil, errors.Join(newErrInvalid("tls configuration"), err)
		}
		opts = append(opts, otlploggrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlploggrpc.New(ctx, opts...)
}
