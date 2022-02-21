package spanmetrics

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	gen "github.com/grafana/tempo/modules/generator/processor"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

type processor struct {
	cfg Config

	spanMetricsCallsTotal      *prometheus.CounterVec
	spanMetricsDurationSeconds *prometheus.HistogramVec

	// for testing
	now func() time.Time
}

func New(cfg Config, tenant string) gen.Processor {
	return &processor{
		cfg: cfg,
		now: time.Now,
	}
}

func (p *processor) Name() string { return Name }

func (p *processor) RegisterMetrics(reg prometheus.Registerer) error {
	labelNames := []string{"service", "span_name", "span_kind", "span_status"}
	if len(p.cfg.Dimensions) > 0 {
		labelNames = append(labelNames, p.cfg.Dimensions...)
	}

	p.spanMetricsCallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "traces",
		Name:      "spanmetrics_calls_total",
		Help:      "Total count of the spans",
	}, labelNames)
	p.spanMetricsDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "traces",
		Name:      "spanmetrics_duration_seconds",
		Help:      "Latency of the spans",
		Buckets:   p.cfg.HistogramBuckets,
	}, labelNames)

	cs := []prometheus.Collector{
		p.spanMetricsCallsTotal,
		p.spanMetricsDurationSeconds,
	}

	for _, c := range cs {
		if err := reg.Register(c); err != nil {
			return err
		}
	}

	return nil
}

func (p *processor) unregisterMetrics(reg prometheus.Registerer) {
	cs := []prometheus.Collector{
		p.spanMetricsCallsTotal,
		p.spanMetricsDurationSeconds,
	}

	for _, c := range cs {
		reg.Unregister(c)
	}
}

func (p *processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "spanmetrics.PushSpans")
	defer span.Finish()

	p.aggregateMetrics(req.Batches)

	return nil
}

func (p *processor) Shutdown(ctx context.Context, reg prometheus.Registerer) error {
	p.unregisterMetrics(reg)
	return nil
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

	labelValues := []string{svcName, span.GetName(), span.GetKind().String(), span.GetStatus().GetCode().String()}

	if len(p.cfg.Dimensions) > 0 {
		// Build additional dimensions
		for _, d := range p.cfg.Dimensions {
			for _, attr := range span.Attributes {
				if d == attr.Key {
					labelValues = append(labelValues, attr.GetValue().GetStringValue())
				}
			}
		}

	}

	p.spanMetricsCallsTotal.WithLabelValues(labelValues...).Inc()
	p.spanMetricsDurationSeconds.WithLabelValues(labelValues...).(prometheus.ExemplarObserver).ObserveWithExemplar(
		latencySeconds, prometheus.Labels{"traceID": tempo_util.TraceIDToHexString(span.TraceId)},
	)
}
