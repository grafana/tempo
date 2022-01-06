package spanmetrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/storage"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
)

const (
	name        = "spanmetrics"
	callsMetric = "calls_total"
)

type processor struct {
	namespace, tenant string

	appender storage.Appender

	mtx   sync.Mutex
	calls map[string]float64

	cache map[string]labels.Labels
}

func New(tenant string, appender storage.Appender) gen.Processor {
	return &processor{
		namespace: "tempo",
		tenant:    tenant,
		appender:  appender,
		calls:     make(map[string]float64),
		cache:     make(map[string]labels.Labels),
	}
}

func (p *processor) Name() string { return name }

func (p *processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) error {
	p.aggregateMetrics(req.Batches)

	return p.collectMetrics()
}

func (p *processor) Shutdown(context.Context) error { return nil }

func (p *processor) aggregateMetrics(resourceSpans []*v1_trace.ResourceSpans) {
	for _, rs := range resourceSpans {
		svcName := getServiceName(rs.Resource)
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
	key := p.buildKey(svcName, span)

	p.mtx.Lock()
	p.calls[key]++
	p.mtx.Unlock()
}

func (p *processor) collectMetrics() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	t := time.Now().Unix()

	var errs error
	for key, count := range p.calls {
		lbls := p.getLabels(key, callsMetric)

		if _, err := p.appender.Append(0, lbls, t, count); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	if errs != nil {
		return errs
	}

	return p.appender.Commit()
}

func (p *processor) buildKey(svcName string, span *v1_trace.Span) string {
	// TODO: add more dimensions
	key := fmt.Sprintf("%s_%s_%s_%s", svcName, span.Name, span.Kind, span.Status)

	p.cacheLabels(key, svcName, span)

	return key
}

// Must be called under lock
func (p *processor) cacheLabels(key string, svcName string, span *v1_trace.Span) {
	p.cache[key] = labels.Labels{
		{Name: "service", Value: svcName},
		{Name: "span_name", Value: span.Name},
		{Name: "span_kind", Value: span.Kind.String()},
		{Name: "span_status", Value: span.Status.Code.String()},
	}
}

// Must be called under lock
func (p *processor) getLabels(key, metricName string) labels.Labels {
	// TODO: check if it doesn't exist?
	lbls := p.cache[key]

	lbls = append(lbls, labels.Label{Name: "__name__", Value: fmt.Sprintf("%s_%s", p.namespace, metricName)})
	lbls = append(lbls, labels.Label{Name: "tenant", Value: p.tenant})

	return lbls
}

func getServiceName(rs *v1_resource.Resource) string {
	for _, attr := range rs.Attributes {
		if attr.Key == semconv.AttributeServiceName {
			return attr.Value.GetStringValue()
		}
	}

	return ""
}
