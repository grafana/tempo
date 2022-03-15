package spanmetrics

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"

	gen "github.com/grafana/tempo/modules/generator/processor"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
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
	if cfg.Dimensions != nil {
		// TODO we should convert keys into valid prometheus labels, i.e. k8s.ip -> k8s_ip
		labels = append(labels, cfg.Dimensions...)
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

func (p *processor) Shutdown(ctx context.Context) {
}

func (p *processor) aggregateMetrics(resourceSpans []*v1_trace.ResourceSpans) {
	for _, rs := range resourceSpans {
		svcName := processor_util.GetServiceName(rs.Resource)
		if svcName == "" {
			continue
		}
		for _, ils := range rs.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				p.aggregateMetricsForSpan(svcName, span)
			}
		}
	}
}

func (p *processor) aggregateMetricsForSpan(svcName string, span *v1_trace.Span) {
	latencySeconds := float64(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano()) / float64(time.Second.Nanoseconds())

	labelValues := make([]string, 0, 4+len(p.cfg.Dimensions))
	labelValues = append(labelValues, svcName, span.GetName(), span.GetKind().String(), span.GetStatus().GetCode().String())

	for _, d := range p.cfg.Dimensions {
		for _, attr := range span.Attributes {
			if d == attr.Key {
				labelValues = append(labelValues, attr.GetValue().GetStringValue())
			}
		}
	}

	registrylabelValues := registry.NewLabelValues(labelValues)

	p.spanMetricsCallsTotal.Inc(registrylabelValues, 1)
	// TODO observe exemplar prometheus.Labels{"traceID": tempo_util.TraceIDToHexString(span.TraceId)}
	p.spanMetricsDurationSeconds.Observe(registrylabelValues, latencySeconds)
}
