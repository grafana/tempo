package generator

import (
	"context"
	"fmt"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"
	"github.com/weaveworks/common/tracing"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	metricSpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_spans_received_total",
		Help:      "The total number of spans received per tenant",
	}, []string{"tenant"})
	metricBytesIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_bytes_received_total",
		Help:      "The total number of proto bytes received per tenant",
	}, []string{"tenant"})
)

type instance struct {
	cfg *Config

	instanceID string
	overrides  *overrides.Overrides

	registry   processor.Registry
	appendable storage.Appendable

	processors []processor.Processor

	metricSpansIngestedTotal prometheus.Counter
	metricBytesIngestedTotal prometheus.Counter
}

func newInstance(cfg *Config, instanceID string, overrides *overrides.Overrides, appendable storage.Appendable) (*instance, error) {
	i := &instance{
		cfg:        cfg,
		instanceID: instanceID,
		overrides:  overrides,

		registry:   processor.NewRegistry(cfg.ExternalLabels),
		appendable: appendable,

		metricSpansIngestedTotal: metricSpansIngested.WithLabelValues(instanceID),
		metricBytesIngestedTotal: metricBytesIngested.WithLabelValues(instanceID),
	}

	// TODO we should build a pipeline based upon the overrides configured
	// TODO when the overrides change we should update all the processors/the pipeline
	spanMetricsProcessor := spanmetrics.New(i.cfg.Processor.SpanMetrics, instanceID)
	serviceGraphProcessor := servicegraphs.New(i.cfg.Processor.ServiceGraphs, instanceID)

	i.processors = []processor.Processor{serviceGraphProcessor, spanMetricsProcessor}

	for _, processor := range i.processors {
		err := processor.RegisterMetrics(i.registry)
		if err != nil {
			return nil, fmt.Errorf("error registering metrics for %s: %w", processor.Name(), err)
		}
	}

	return i, nil
}

func (i *instance) pushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	i.updateMetrics(req)

	for _, processor := range i.processors {
		if err := processor.PushSpans(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

func (i *instance) updateMetrics(req *tempopb.PushSpansRequest) {
	size := 0
	spanCount := 0
	for _, b := range req.Batches {
		size += b.Size()
		for _, ils := range b.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}
	}
	i.metricBytesIngestedTotal.Add(float64(size))
	i.metricSpansIngestedTotal.Add(float64(spanCount))
}

func (i *instance) collectAndPushMetrics(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.collectAndPushMetrics")
	defer span.Finish()

	traceID, _ := tracing.ExtractTraceID(ctx)
	level.Info(log.Logger).Log("msg", "collecting metrics", "tenant", i.instanceID, "traceID", traceID)

	appender := i.appendable.Appender(ctx)

	err := i.registry.Gather(appender)
	if err != nil {
		return err
	}

	return appender.Commit()
}

// shutdown stops the instance and flushes any remaining data. After shutdown
// is called PushSpans should not be called anymore.
func (i *instance) shutdown(ctx context.Context) error {
	// TODO should we set a boolean to refuse push request once this is called?

	err := i.collectAndPushMetrics(ctx)
	if err != nil {
		level.Error(log.Logger).Log("msg", "collecting metrics failed at shutdown", "tenant", i.instanceID, "err", err)
	}

	for _, processor := range i.processors {
		processor.UnregisterMetrics(i.registry)

		err := processor.Shutdown(ctx)
		if err != nil {
			level.Warn(log.Logger).Log("msg", "failed to shutdown processor", "processor", processor.Name(), "err", err)
		}
	}

	return nil
}
