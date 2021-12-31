package collector

import (
	"context"
	"fmt"
	"strings"

	promexporter "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/cortexproject/cortex/pkg/util/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/generator/exporter/remotewriteexporter"
	"github.com/grafana/tempo/modules/generator/pushreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanmetricsprocessor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/storage"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/external/obsreportconfig"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/service/external/builder"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Collector struct {
	// mut         sync.Mutex
	cfg         interface{}
	logger      kitlog.Logger
	metricViews []*view.View

	// TODO: add concurrency? ⚡
	pushChan chan pdata.Traces

	appendable storage.Appendable

	exporters  builder.Exporters
	processors builder.BuiltPipelines
	receivers  builder.Receivers
}

func New(ctx context.Context, reg prometheus.Registerer, appendable storage.Appendable) (*Collector, error) {
	var err error
	instance := &Collector{
		pushChan:   make(chan pdata.Traces),
		appendable: appendable,
	}
	instance.logger = log.Logger
	instance.metricViews, err = newMetricViews(reg)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric views: %w", err)
	}

	// TODO: Apply config at runtime, so instances can be created dynamically and updated 🏃
	if err := instance.buildAndStartPipeline(ctx); err != nil {
		return nil, fmt.Errorf("failed to build and start pipeline: %w", err)
	}

	return instance, nil
}

func (c *Collector) Shutdown(ctx context.Context) error {
	deps := []func(ctx context.Context) error{
		c.receivers.ShutdownAll,
		c.processors.ShutdownProcessors,
		c.exporters.ShutdownAll,
	}

	for _, d := range deps {
		if err := d(ctx); err != nil {
			level.Warn(c.logger).Log("msg", "failed to shutdown component", "err", err)
		}
	}

	view.Unregister(c.metricViews...)

	return nil
}

func (c *Collector) PushSpans(ctx context.Context, td pdata.Traces) error {
	select {
	case c.pushChan <- td:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Collector) buildAndStartPipeline(ctx context.Context) error {
	// Michael, this is so no right.
	// TODO: Make less brittle.
	conf := `
processors:
  spanmetrics:
    metrics_exporter: remote_write
    latency_histogram_buckets: [1ms, 10ms, 100ms]
exporters:
  noop:
  remote_write:
    namespace: tempo
  prometheus:
    endpoint: localhost:9090
receivers:
  push:
  noop:
service:
  pipelines:
    traces:
      exporters: [noop]
      processors: [spanmetrics]
      receivers: [push]
    metrics/spanmetrics:
      exporters: [remote_write]
      receivers: [noop]
`

	var cfg map[string]interface{}
	if err := yaml.NewDecoder(strings.NewReader(conf)).Decode(&cfg); err != nil {
		panic("could not decode config: " + err.Error())
	}

	factories, err := c.getFactories()
	if err != nil {
		return fmt.Errorf("failed to get factories: %w", err)
	}

	configMap := config.NewMapFromStringMap(cfg)
	cfgUnmarshaler := configunmarshaler.NewDefault()
	otelCfg, err := cfgUnmarshaler.Unmarshal(configMap, factories)
	if err != nil {
		return fmt.Errorf("failed to make otel config: %w", err)
	}

	var buildInfo component.BuildInfo

	settings := component.TelemetrySettings{
		Logger:         zap.NewNop(),
		TracerProvider: trace.NewNoopTracerProvider(),
		MeterProvider:  metric.NewNoopMeterProvider(),
	}

	c.exporters, err = builder.BuildExporters(settings, buildInfo, otelCfg, factories.Exporters)
	if err != nil {
		return fmt.Errorf("failed to build exporters: %w", err)
	}
	if err := c.exporters.StartAll(ctx, c); err != nil {
		return fmt.Errorf("failed to start exporters: %w", err)
	}

	c.processors, err = builder.BuildPipelines(settings, buildInfo, otelCfg, c.exporters, factories.Processors)
	if err != nil {
		return fmt.Errorf("failed to build processors: %w", err)
	}
	if err := c.processors.StartProcessors(ctx, c); err != nil {
		return fmt.Errorf("failed to start processors: %w", err)
	}

	c.receivers, err = builder.BuildReceivers(settings, buildInfo, otelCfg, c.processors, factories.Receivers)
	if err != nil {
		return fmt.Errorf("failed to build receivers: %w", err)
	}
	if err := c.receivers.StartAll(ctx, c); err != nil {
		return fmt.Errorf("failed to start receivers: %w", err)
	}

	return nil
}

func (c *Collector) ReportFatalError(err error) {
	level.Error(c.logger).Log("fatal error reported", zap.Error(err))
}

func (c *Collector) GetFactory(component.Kind, config.Type) component.Factory { return nil }

func (c *Collector) GetExtensions() map[config.ComponentID]component.Extension { return nil }

func (c *Collector) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	return c.exporters.ToMapByDataType()
}

func newMetricViews(reg prometheus.Registerer) ([]*view.View, error) {
	obsMetrics := obsreportconfig.Configure(configtelemetry.LevelBasic)
	err := view.Register(obsMetrics.Views...)
	if err != nil {
		return nil, fmt.Errorf("failed to register views: %w", err)
	}

	pe, err := promexporter.NewExporter(promexporter.Options{
		Namespace:  "tempo",
		Registerer: reg,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	view.RegisterExporter(pe)

	return obsMetrics.Views, nil
}

func (c *Collector) getFactories() (component.Factories, error) {
	extensionsFactory, err := component.MakeExtensionFactoryMap()
	if err != nil {
		return component.Factories{}, fmt.Errorf("failed to make extension factory map: %w", err)
	}

	receiversFactory, err := component.MakeReceiverFactoryMap(
		newNoopReceiverFactory(),
		pushreceiver.NewFactory(c.pushChan),
	)
	if err != nil {
		return component.Factories{}, fmt.Errorf("failed to make receiver factory map: %w", err)
	}

	exportersFactory, err := component.MakeExporterFactoryMap(
		newNoopExporterFactory(),
		remotewriteexporter.NewFactory(c.appendable),
		prometheusexporter.NewFactory(),
	)
	if err != nil {
		return component.Factories{}, fmt.Errorf("failed to make exporter factory map: %w", err)
	}

	processorsFactory, err := component.MakeProcessorFactoryMap(
		spanmetricsprocessor.NewFactory(),
	)
	if err != nil {
		return component.Factories{}, fmt.Errorf("failed to make processor factory map: %w", err)
	}

	return component.Factories{
		Extensions: extensionsFactory,
		Receivers:  receiversFactory,
		Processors: processorsFactory,
		Exporters:  exportersFactory,
	}, nil
}
