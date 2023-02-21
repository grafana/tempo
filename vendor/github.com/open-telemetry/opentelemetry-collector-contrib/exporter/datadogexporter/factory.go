// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datadogexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/otlp/model/source"
	"github.com/DataDog/datadog-agent/pkg/trace/agent"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry"
)

const (
	// typeStr is the type of the exporter
	typeStr = "datadog"
)

var mertricExportNativeClientFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"exporter.datadogexporter.metricexportnativeclient",
	featuregate.StageBeta,
	featuregate.WithRegisterDescription("When enabled, metric export in datadogexporter uses native Datadog client APIs instead of Zorkian APIs."),
)

// isMetricExportV2Enabled returns true if metric export in datadogexporter uses native Datadog client APIs, false if it uses Zorkian APIs
func isMetricExportV2Enabled() bool {
	return mertricExportNativeClientFeatureGate.IsEnabled()
}

// enableNativeMetricExport switches metric export to call native Datadog APIs instead of Zorkian APIs.
func enableNativeMetricExport() error {
	return featuregate.GlobalRegistry().Set(mertricExportNativeClientFeatureGate.ID(), true)
}

// enableZorkianMetricExport switches metric export to call Zorkian APIs instead of native Datadog APIs.
func enableZorkianMetricExport() error {
	return featuregate.GlobalRegistry().Set(mertricExportNativeClientFeatureGate.ID(), false)
}

type factory struct {
	onceMetadata sync.Once

	onceProvider   sync.Once
	sourceProvider source.Provider
	providerErr    error

	wg sync.WaitGroup // waits for agent to exit

	registry *featuregate.Registry
}

func (f *factory) SourceProvider(set component.TelemetrySettings, configHostname string) (source.Provider, error) {
	f.onceProvider.Do(func() {
		f.sourceProvider, f.providerErr = metadata.GetSourceProvider(set, configHostname)
	})
	return f.sourceProvider, f.providerErr
}

func (f *factory) TraceAgent(ctx context.Context, params exporter.CreateSettings, cfg *Config, sourceProvider source.Provider) (*agent.Agent, error) {
	agnt, err := newTraceAgent(ctx, params, cfg, sourceProvider)
	if err != nil {
		return nil, err
	}
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		agnt.Run()
	}()
	return agnt, nil
}

func newFactoryWithRegistry(registry *featuregate.Registry) exporter.Factory {
	f := &factory{registry: registry}
	return exporter.NewFactory(
		typeStr,
		f.createDefaultConfig,
		exporter.WithMetrics(f.createMetricsExporter, component.StabilityLevelBeta),
		exporter.WithTraces(f.createTracesExporter, component.StabilityLevelBeta),
		exporter.WithLogs(f.createLogsExporter, component.StabilityLevelAlpha),
	)
}

// NewFactory creates a Datadog exporter factory
func NewFactory() exporter.Factory {
	return newFactoryWithRegistry(featuregate.GlobalRegistry())
}

func defaulttimeoutSettings() exporterhelper.TimeoutSettings {
	return exporterhelper.TimeoutSettings{
		Timeout: 15 * time.Second,
	}
}

// createDefaultConfig creates the default exporter configuration
func (f *factory) createDefaultConfig() component.Config {
	hostnameSource := HostnameSourceFirstResource
	if metadata.HostnamePreviewFeatureGate.IsEnabled() {
		hostnameSource = HostnameSourceConfigOrSystem
	}

	return &Config{
		TimeoutSettings: defaulttimeoutSettings(),
		RetrySettings:   exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:   exporterhelper.NewDefaultQueueSettings(),

		API: APIConfig{
			Site: "datadoghq.com",
		},

		Metrics: MetricsConfig{
			TCPAddr: confignet.TCPAddr{
				Endpoint: "https://api.datadoghq.com",
			},
			DeltaTTL: 3600,
			ExporterConfig: MetricsExporterConfig{
				ResourceAttributesAsTags:           false,
				InstrumentationScopeMetadataAsTags: false,
			},
			HistConfig: HistogramConfig{
				Mode:         "distributions",
				SendCountSum: false,
			},
			SumConfig: SumConfig{
				CumulativeMonotonicMode: CumulativeMonotonicSumModeToDelta,
			},
			SummaryConfig: SummaryConfig{
				Mode: SummaryModeGauges,
			},
		},

		Traces: TracesConfig{
			TCPAddr: confignet.TCPAddr{
				Endpoint: "https://trace.agent.datadoghq.com",
			},
			IgnoreResources: []string{},
		},

		Logs: LogsConfig{
			TCPAddr: confignet.TCPAddr{
				Endpoint: "https://http-intake.logs.datadoghq.com",
			},
		},

		HostMetadata: HostMetadataConfig{
			Enabled:        true,
			HostnameSource: hostnameSource,
		},
	}
}

// checkAndCastConfig checks the configuration type and its warnings, and casts it to
// the Datadog Config struct.
func checkAndCastConfig(c component.Config) *Config {
	cfg, ok := c.(*Config)
	if !ok {
		panic("programming error: config structure is not of type *datadogexporter.Config")
	}
	return cfg
}

// createMetricsExporter creates a metrics exporter based on this config.
func (f *factory) createMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	c component.Config,
) (exporter.Metrics, error) {
	cfg := checkAndCastConfig(c)

	hostProvider, err := f.SourceProvider(set.TelemetrySettings, cfg.Hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to build hostname provider: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	// cancel() runs on shutdown
	var pushMetricsFn consumer.ConsumeMetricsFunc
	traceagent, err := f.TraceAgent(ctx, set, cfg, hostProvider)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start trace-agent: %w", err)
	}
	if cfg.OnlyMetadata {
		pushMetricsFn = func(_ context.Context, md pmetric.Metrics) error {
			// only sending metadata use only metrics
			f.onceMetadata.Do(func() {
				attrs := pcommon.NewMap()
				if md.ResourceMetrics().Len() > 0 {
					attrs = md.ResourceMetrics().At(0).Resource().Attributes()
				}
				go metadata.Pusher(ctx, set, newMetadataConfigfromConfig(cfg), hostProvider, attrs)
			})

			return nil
		}
	} else {
		exp, metricsErr := newMetricsExporter(ctx, set, cfg, &f.onceMetadata, hostProvider, traceagent)
		if metricsErr != nil {
			cancel()    // first cancel context
			f.wg.Wait() // then wait for shutdown
			return nil, metricsErr
		}
		pushMetricsFn = exp.PushMetricsDataScrubbed
	}

	exporter, err := exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		pushMetricsFn,
		// explicitly disable since we rely on http.Client timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0 * time.Second}),
		// We use our own custom mechanism for retries, since we hit several endpoints.
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithShutdown(func(context.Context) error {
			cancel()
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	return resourcetotelemetry.WrapMetricsExporter(
		resourcetotelemetry.Settings{Enabled: cfg.Metrics.ExporterConfig.ResourceAttributesAsTags}, exporter), nil
}

// createTracesExporter creates a trace exporter based on this config.
func (f *factory) createTracesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	c component.Config,
) (exporter.Traces, error) {
	cfg := checkAndCastConfig(c)

	var (
		pusher consumer.ConsumeTracesFunc
		stop   component.ShutdownFunc
	)

	hostProvider, err := f.SourceProvider(set.TelemetrySettings, cfg.Hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to build hostname provider: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	// cancel() runs on shutdown
	traceagent, err := f.TraceAgent(ctx, set, cfg, hostProvider)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start trace-agent: %w", err)
	}
	if cfg.OnlyMetadata {
		// only host metadata needs to be sent, once.
		pusher = func(_ context.Context, td ptrace.Traces) error {
			f.onceMetadata.Do(func() {
				attrs := pcommon.NewMap()
				if td.ResourceSpans().Len() > 0 {
					attrs = td.ResourceSpans().At(0).Resource().Attributes()
				}
				go metadata.Pusher(ctx, set, newMetadataConfigfromConfig(cfg), hostProvider, attrs)
			})
			return nil
		}
		stop = func(context.Context) error {
			cancel()
			return nil
		}
	} else {
		tracex, err2 := newTracesExporter(ctx, set, cfg, &f.onceMetadata, hostProvider, traceagent)
		if err2 != nil {
			cancel()
			return nil, err2
		}
		pusher = tracex.consumeTraces
		stop = func(context.Context) error {
			cancel()    // first cancel context
			f.wg.Wait() // then wait for shutdown
			return nil
		}
	}

	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		pusher,
		// explicitly disable since we rely on http.Client timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0 * time.Second}),
		// We don't do retries on traces because of deduping concerns on APM Events.
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithShutdown(stop),
	)
}

// createLogsExporter creates a logs exporter based on the config.
func (f *factory) createLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	c component.Config,
) (exporter.Logs, error) {
	cfg := checkAndCastConfig(c)

	var pusher consumer.ConsumeLogsFunc
	hostProvider, err := f.SourceProvider(set.TelemetrySettings, cfg.Hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to build hostname provider: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	// cancel() runs on shutdown
	if cfg.OnlyMetadata {
		// only host metadata needs to be sent, once.
		pusher = func(_ context.Context, td plog.Logs) error {
			f.onceMetadata.Do(func() {
				attrs := pcommon.NewMap()
				go metadata.Pusher(ctx, set, newMetadataConfigfromConfig(cfg), hostProvider, attrs)
			})
			return nil
		}
	} else {
		exp, err := newLogsExporter(ctx, set, cfg, &f.onceMetadata, hostProvider)
		if err != nil {
			cancel()
			return nil, err
		}
		pusher = exp.consumeLogs
	}
	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		pusher,
		// explicitly disable since we rely on http.Client timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0 * time.Second}),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithShutdown(func(context.Context) error {
			cancel()
			return nil
		}),
	)
}
