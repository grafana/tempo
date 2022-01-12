package generator

import (
	"context"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"

	"github.com/grafana/tempo/modules/generator/processor"
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
	instanceID string
	overrides  *overrides.Overrides

	registerer prometheus.Registerer
	appendable storage.Appendable

	processors []processor.Processor

	metricSpansIngestedTotal prometheus.Counter
	metricBytesIngestedTotal prometheus.Counter
}

func newInstance(instanceID string, overrides *overrides.Overrides, userMetricsRegisterer prometheus.Registerer, appendable storage.Appendable) (*instance, error) {
	i := &instance{
		instanceID: instanceID,
		overrides:  overrides,

		registerer: userMetricsRegisterer,
		appendable: appendable,

		metricSpansIngestedTotal: metricSpansIngested.WithLabelValues(instanceID),
		metricBytesIngestedTotal: metricBytesIngested.WithLabelValues(instanceID),
	}

	// TODO we should build a pipeline based upon the overrides configured
	// TODO when the overrides change we should update all the processors/the pipeline
	spanMetricsProcessor := spanmetrics.New()

	i.processors = []processor.Processor{spanMetricsProcessor}

	return i, nil
}

func (i *instance) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	size := 0
	spanCount := 0
	for _, b := range req.Batches {
		size += b.Size()
		for _, ils := range b.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}
	}
	if spanCount == 0 {
		return nil
	}
	i.metricBytesIngestedTotal.Add(float64(size))
	i.metricSpansIngestedTotal.Add(float64(spanCount))

	for _, processor := range i.processors {
		if err := processor.PushSpans(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

func (i *instance) collectAndPushMetrics(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.collectAndPushMetrics")
	defer span.Finish()

	level.Debug(log.Logger).Log("msg", "collecting metrics", "tenant", i.instanceID)

	appender := i.appendable.Appender(ctx)

	for _, processor := range i.processors {
		err := processor.CollectMetrics(ctx, appender)
		if err != nil {
			return err
		}
	}

	return appender.Commit()
}

// Shutdown stops the instance and flushes any remaining data. After shutdown
// is called PushSpans should not be called anymore.
func (i *instance) Shutdown(ctx context.Context) error {
	// TODO should we set a boolean to refuse push request once this is called?

	for _, processor := range i.processors {
		err := processor.Shutdown(ctx)
		if err != nil {
			level.Warn(log.Logger).Log("msg", "failed to shutdown processor", "processor", processor.Name(), "err", err)
		}
	}

	return nil
}
