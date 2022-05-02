package spanmetrics

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/prometheus/util/strutil"

	gen "github.com/grafana/tempo/modules/generator/processor"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

const (
	metricCallsTotal      = "traces_spanmetrics_calls_total"
	metricDurationSeconds = "traces_spanmetrics_duration_seconds"
)

type processor struct {
	cfg Config

	spanMetricsCallsTotal      registry.Counter
	spanMetricsDurationSeconds registry.Histogram

	// for testing
	now func() time.Time
}

func New(cfg Config, registry registry.Registry) gen.Processor {
	labels := []string{"service", "span_name", "span_kind", "span_status"}
	for _, d := range cfg.Dimensions {
		labels = append(labels, strutil.SanitizeLabelName(d))
	}

	return &processor{
		cfg:                        cfg,
		spanMetricsCallsTotal:      registry.NewCounter(metricCallsTotal, labels),
		spanMetricsDurationSeconds: registry.NewHistogram(metricDurationSeconds, labels, cfg.HistogramBuckets),
		now:                        time.Now,
	}
}

func (p *processor) Name() string { return Name }

func (p *processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	span, _ := opentracing.StartSpanFromContext(ctx, "spanmetrics.PushSpans")
	defer span.Finish()

	p.aggregateMetrics(req.Batches)
}

func (p *processor) Shutdown(_ context.Context) {
}

func (p *processor) aggregateMetrics(resourceSpans []*v1_trace.ResourceSpans) {
	for _, rs := range resourceSpans {
		// already extract service name, so we only have to do it once per batch of spans
		svcName, _ := processor_util.FindServiceName(rs.Resource.Attributes)

		for _, ils := range rs.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				p.aggregateMetricsForSpan(svcName, rs.Resource, span)
			}
		}
	}
}

func (p *processor) aggregateMetricsForSpan(svcName string, rs *v1.Resource, span *v1_trace.Span) {
	latencySeconds := float64(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano()) / float64(time.Second.Nanoseconds())

	labelValues := make([]string, 0, 4+len(p.cfg.Dimensions))
	labelValues = append(labelValues, svcName, span.GetName(), span.GetKind().String(), span.GetStatus().GetCode().String())

	for _, d := range p.cfg.Dimensions {
		value, _ := processor_util.FindAttributeValue(d, rs.Attributes, span.Attributes)
		labelValues = append(labelValues, value)
	}

	registryLabelValues := registry.NewLabelValues(labelValues)

	p.spanMetricsCallsTotal.Inc(registryLabelValues, 1)
	p.spanMetricsDurationSeconds.ObserveWithExemplar(registryLabelValues, latencySeconds, tempo_util.TraceIDToHexString(span.TraceId))
}
