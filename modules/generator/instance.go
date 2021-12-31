package generator

import (
	"context"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/generator/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	metricSpansReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_spans_received_total",
		Help:      "The total number of spans received.",
	}, []string{"tenant"})
)

type instance struct {
	instanceID string
	overrides  *overrides.Overrides

	registerer prometheus.Registerer
	appendable storage.Appendable

	processors []processor.Processor

	metricSpansReceivedTotal prometheus.Counter

	collector *collector.Collector
}

func newInstance(instanceID string, overrides *overrides.Overrides, userMetricsRegisterer prometheus.Registerer, appendable storage.Appendable) (*instance, error) {
	i := &instance{
		instanceID: instanceID,
		overrides:  overrides,

		registerer: userMetricsRegisterer,
		appendable: appendable,

		metricSpansReceivedTotal: metricSpansReceivedTotal.WithLabelValues(instanceID),
	}

	var err error
	i.collector, err = collector.New(context.Background(), userMetricsRegisterer, appendable)
	if err != nil {
		return nil, err
	}

	// TODO we should build a pipeline based upon the overrides configured
	// TODO when the overrides change we should update all the processors/the pipeline
	serviceGraphProcessor, err := processor.NewServiceGraphProcessor(i.registerer)
	if err != nil {
		return nil, err
	}
	i.processors = []processor.Processor{serviceGraphProcessor}

	return i, nil
}

func (i *instance) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	i.metricSpansReceivedTotal.Inc()

	otlpTraces, err := convertToOTLP(ctx, req)
	if err != nil {
		return err
	}

	if err := i.collector.PushSpans(ctx, otlpTraces); err != nil {
		return err
	}

	for _, processor := range i.processors {
		err = processor.ConsumeTraces(ctx, otlpTraces)
		if err != nil {
			return err
		}
	}

	return nil
}

func convertToOTLP(_ context.Context, req *tempopb.PushSpansRequest) (pdata.Traces, error) {
	// We do the reverse of the Distributor here: convert to bytes and back to
	// OTLP proto. This is unfortunate for efficiency, but it works around the
	// otel-collector internalization of otel-proto which Tempo also uses.

	trace := tempopb.Trace{}
	trace.Batches = req.Batches

	bytes, err := trace.Marshal()
	if err != nil {
		return pdata.Traces{}, err
	}

	return otlp.NewProtobufTracesUnmarshaler().UnmarshalTraces(bytes)
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
