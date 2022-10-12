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
	metricDurationSeconds = "traces_spanmetrics_latency"
	metricSizeTotal       = "traces_spanmetrics_size_total"
)

type Processor struct {
	Cfg Config

	spanMetricsCallsTotal      registry.Counter
	spanMetricsDurationSeconds registry.Histogram
	spanMetricsSizeTotal       registry.Counter

	// for testing
	now func() time.Time
}

func New(cfg Config, registry registry.Registry) gen.Processor {
	labels := []string{"service", "span_name", "span_kind", "status_code", "status_message"}
	for _, d := range cfg.Dimensions {
		labels = append(labels, strutil.SanitizeLabelName(d))
	}

	return &Processor{
		Cfg:                        cfg,
		spanMetricsCallsTotal:      registry.NewCounter(metricCallsTotal, labels),
		spanMetricsDurationSeconds: registry.NewHistogram(metricDurationSeconds, labels, cfg.HistogramBuckets),
		spanMetricsSizeTotal:       registry.NewCounter(metricSizeTotal, labels),
		now:                        time.Now,
	}
}

func (p *Processor) Name() string {
	return Name
}

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	span, _ := opentracing.StartSpanFromContext(ctx, "spanmetrics.PushSpans")
	defer span.Finish()

	p.aggregateMetrics(req.Batches)
}

func (p *Processor) Shutdown(_ context.Context) {
}

func (p *Processor) aggregateMetrics(resourceSpans []*v1_trace.ResourceSpans) {
	for _, rs := range resourceSpans {
		// already extract service name, so we only have to do it once per batch of spans
		svcName, _ := processor_util.FindServiceName(rs.Resource.Attributes)

		for _, ils := range rs.ScopeSpans {
			for _, span := range ils.Spans {
				p.aggregateMetricsForSpan(svcName, rs.Resource, span)
			}
		}
	}
}

func (p *Processor) aggregateMetricsForSpan(svcName string, rs *v1.Resource, span *v1_trace.Span) {
	latencySeconds := float64(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano()) / float64(time.Second.Nanoseconds())

	labelValues := make([]string, 0, 4+len(p.Cfg.Dimensions))
	labelValues = append(
		labelValues,
		svcName,
		span.GetName(),
		span.GetKind().String(),
		span.GetStatus().GetCode().String(),
		span.GetStatus().GetMessage())

	for _, d := range p.Cfg.Dimensions {
		value, _ := processor_util.FindAttributeValue(d, rs.Attributes, span.Attributes)
		labelValues = append(labelValues, value)
	}

	registryLabelValues := registry.NewLabelValues(labelValues)

	p.spanMetricsCallsTotal.Inc(registryLabelValues, 1)
	p.spanMetricsSizeTotal.Inc(registryLabelValues, float64(span.Size()))
	p.spanMetricsDurationSeconds.ObserveWithExemplar(registryLabelValues, latencySeconds, tempo_util.TraceIDToHexString(span.TraceId))
}
