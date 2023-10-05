// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"fmt"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
)

// Config defines the configurable settings for service telemetry.
type Config struct {
	Logs    LogsConfig    `mapstructure:"logs"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Traces  TracesConfig  `mapstructure:"traces"`

	// Resource specifies user-defined attributes to include with all emitted telemetry.
	// Note that some attributes are added automatically (e.g. service.version) even
	// if they are not specified here. In order to suppress such attributes the
	// attribute must be specified in this map with null YAML value (nil string pointer).
	Resource map[string]*string `mapstructure:"resource"`
}

// LogsConfig defines the configurable settings for service telemetry logs.
// This MUST be compatible with zap.Config. Cannot use directly zap.Config because
// the collector uses mapstructure and not yaml tags.
type LogsConfig struct {
	// Level is the minimum enabled logging level.
	// (default = "INFO")
	Level zapcore.Level `mapstructure:"level"`

	// Development puts the logger in development mode, which changes the
	// behavior of DPanicLevel and takes stacktraces more liberally.
	// (default = false)
	Development bool `mapstructure:"development"`

	// Encoding sets the logger's encoding.
	// Example values are "json", "console".
	Encoding string `mapstructure:"encoding"`

	// DisableCaller stops annotating logs with the calling function's file
	// name and line number. By default, all logs are annotated.
	// (default = false)
	DisableCaller bool `mapstructure:"disable_caller"`

	// DisableStacktrace completely disables automatic stacktrace capturing. By
	// default, stacktraces are captured for WarnLevel and above logs in
	// development and ErrorLevel and above in production.
	// (default = false)
	DisableStacktrace bool `mapstructure:"disable_stacktrace"`

	// Sampling sets a sampling policy. A nil SamplingConfig disables sampling.
	Sampling *LogsSamplingConfig `mapstructure:"sampling"`

	// OutputPaths is a list of URLs or file paths to write logging output to.
	// The URLs could only be with "file" schema or without schema.
	// The URLs with "file" schema must be an absolute path.
	// The URLs without schema are treated as local file paths.
	// "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr.
	// see details at Open in zap/writer.go.
	// (default = ["stderr"])
	OutputPaths []string `mapstructure:"output_paths"`

	// ErrorOutputPaths is a list of URLs or file paths to write zap internal logger errors to.
	// The URLs could only be with "file" schema or without schema.
	// The URLs with "file" schema must use absolute paths.
	// The URLs without schema are treated as local file paths.
	// "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr.
	// see details at Open in zap/writer.go.
	//
	// Note that this setting only affects the zap internal logger errors.
	// (default = ["stderr"])
	ErrorOutputPaths []string `mapstructure:"error_output_paths"`

	// InitialFields is a collection of fields to add to the root logger.
	// Example:
	//
	// 		initial_fields:
	//	   		foo: "bar"
	//
	// By default, there is no initial field.
	InitialFields map[string]any `mapstructure:"initial_fields"`
}

// LogsSamplingConfig sets a sampling strategy for the logger. Sampling caps the
// global CPU and I/O load that logging puts on your process while attempting
// to preserve a representative subset of your logs.
type LogsSamplingConfig struct {
	Initial    int `mapstructure:"initial"`
	Thereafter int `mapstructure:"thereafter"`
}

// MetricsConfig exposes the common Telemetry configuration for one component.
// Experimental: *NOTE* this structure is subject to change or removal in the future.
type MetricsConfig struct {
	// Level is the level of telemetry metrics, the possible values are:
	//  - "none" indicates that no telemetry data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	//  - "normal" adds some other indicators on top of basic.
	//  - "detailed" adds dimensions and views to the previous levels.
	Level configtelemetry.Level `mapstructure:"level"`

	// Address is the [address]:port that metrics exposition should be bound to.
	Address string `mapstructure:"address"`

	// Readers allow configuration of metric readers to emit metrics to
	// any number of supported backends.
	Readers []MetricReader `mapstructure:"readers"`
}

// TracesConfig exposes the common Telemetry configuration for collector's internal spans.
// Experimental: *NOTE* this structure is subject to change or removal in the future.
type TracesConfig struct {
	// Propagators is a list of TextMapPropagators from the supported propagators list. Currently,
	// tracecontext and  b3 are supported. By default, the value is set to empty list and
	// context propagation is disabled.
	Propagators []string `mapstructure:"propagators"`
	// Processors allow configuration of span processors to emit spans to
	// any number of suported backends.
	Processors []SpanProcessor `mapstructure:"processors"`
}

// Validate checks whether the current configuration is valid
func (c *Config) Validate() error {
	// Check when service telemetry metric level is not none, the metrics address should not be empty
	if c.Metrics.Level != configtelemetry.LevelNone && c.Metrics.Address == "" && len(c.Metrics.Readers) == 0 {
		return fmt.Errorf("collector telemetry metric address or reader should exist when metric level is not none")
	}

	return nil
}

func (sp *SpanProcessor) Unmarshal(conf *confmap.Conf) error {
	if !obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate.IsEnabled() {
		// only unmarshal if feature gate is enabled
		return nil
	}

	if conf == nil {
		return nil
	}

	if err := conf.Unmarshal(sp); err != nil {
		return fmt.Errorf("invalid span processor configuration: %w", err)
	}

	if sp.Batch != nil {
		return sp.Batch.Exporter.Validate()
	}
	return fmt.Errorf("unsupported span processor type %s", conf.AllKeys())
}

// Validate checks for valid exporters to be configured for the SpanExporter
func (se *SpanExporter) Validate() error {
	if se.Console == nil && se.Otlp == nil {
		return fmt.Errorf("invalid exporter configuration")
	}
	return nil
}

func (mr *MetricReader) Unmarshal(conf *confmap.Conf) error {
	if !obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate.IsEnabled() {
		// only unmarshal if feature gate is enabled
		return nil
	}

	if conf == nil {
		return nil
	}

	if err := conf.Unmarshal(mr); err != nil {
		return fmt.Errorf("invalid metric reader configuration: %w", err)
	}

	if mr.Pull != nil {
		return mr.Pull.Validate()
	}
	if mr.Periodic != nil {
		return mr.Periodic.Validate()
	}

	return fmt.Errorf("unsupported metric reader type %s", conf.AllKeys())
}

// Validate checks for valid exporters to be configured for the PullMetricReader
func (pmr *PullMetricReader) Validate() error {
	if pmr.Exporter.Prometheus == nil {
		return fmt.Errorf("invalid exporter configuration")
	}
	return nil
}

// Validate checks for valid exporters to be configured for the PeriodicMetricReader
func (pmr *PeriodicMetricReader) Validate() error {
	if pmr.Exporter.Otlp == nil && pmr.Exporter.Console == nil {
		return fmt.Errorf("invalid exporter configuration")
	}
	return nil
}
