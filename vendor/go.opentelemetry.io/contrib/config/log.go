// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func loggerProvider(cfg configOptions, res *resource.Resource) (log.LoggerProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.LoggerProvider == nil {
		return noop.NewLoggerProvider(), noopShutdown, nil
	}
	opts := []sdklog.LoggerProviderOption{
		sdklog.WithResource(res),
	}
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
		return nil, errors.New("must not specify multiple log processor type")
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
	return nil, fmt.Errorf("unsupported log processor type, must be one of simple or batch")
}

func logExporter(ctx context.Context, exporter LogRecordExporter) (sdklog.Exporter, error) {
	if exporter.Console != nil && exporter.OTLP != nil {
		return nil, errors.New("must not specify multiple exporters")
	}

	if exporter.Console != nil {
		return stdoutlog.New(
			stdoutlog.WithPrettyPrint(),
		)
	}

	if exporter.OTLP != nil {
		switch exporter.OTLP.Protocol {
		case protocolProtobufHTTP:
			return otlpHTTPLogExporter(ctx, exporter.OTLP)
		default:
			return nil, fmt.Errorf("unsupported protocol %q", exporter.OTLP.Protocol)
		}
	}
	return nil, errors.New("no valid log exporter")
}

func batchLogProcessor(blp *BatchLogRecordProcessor, exp sdklog.Exporter) (*sdklog.BatchProcessor, error) {
	var opts []sdklog.BatchProcessorOption
	if blp.ExportTimeout != nil {
		if *blp.ExportTimeout < 0 {
			return nil, fmt.Errorf("invalid export timeout %d", *blp.ExportTimeout)
		}
		opts = append(opts, sdklog.WithExportTimeout(time.Millisecond*time.Duration(*blp.ExportTimeout)))
	}
	if blp.MaxExportBatchSize != nil {
		if *blp.MaxExportBatchSize < 0 {
			return nil, fmt.Errorf("invalid batch size %d", *blp.MaxExportBatchSize)
		}
		opts = append(opts, sdklog.WithExportMaxBatchSize(*blp.MaxExportBatchSize))
	}
	if blp.MaxQueueSize != nil {
		if *blp.MaxQueueSize < 0 {
			return nil, fmt.Errorf("invalid queue size %d", *blp.MaxQueueSize)
		}
		opts = append(opts, sdklog.WithMaxQueueSize(*blp.MaxQueueSize))
	}

	if blp.ScheduleDelay != nil {
		if *blp.ScheduleDelay < 0 {
			return nil, fmt.Errorf("invalid schedule delay %d", *blp.ScheduleDelay)
		}
		opts = append(opts, sdklog.WithExportInterval(time.Millisecond*time.Duration(*blp.ScheduleDelay)))
	}

	return sdklog.NewBatchProcessor(exp, opts...), nil
}

func otlpHTTPLogExporter(ctx context.Context, otlpConfig *OTLP) (sdklog.Exporter, error) {
	var opts []otlploghttp.Option

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(otlpConfig.Endpoint)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlploghttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		if len(u.Path) > 0 {
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
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlploghttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(otlpConfig.Headers))
	}

	return otlploghttp.New(ctx, opts...)
}
