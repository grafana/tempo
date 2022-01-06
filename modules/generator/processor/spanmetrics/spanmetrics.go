package spanmetrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
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

	// TODO: pass storage.Appender instead?
	//  appendable.Appender(ctx) creates a new Appender for every push.
	appendable storage.Appendable

	// TODO: possibly split mutex into two: one for the metrics and one for the cache.
	//  cache's mutex should be RWMutex.
	mtx sync.Mutex
	// TODO: need a mechanism to clean up inactive series,
	//  otherwise this is unbounded memory usage.
	calls               map[string]float64
	latencyCount        map[string]float64
	latencySum          map[string]float64
	latencyBucketCounts map[string][]float64
	latencyBuckets      []float64
	cache               map[string]labels.Labels
}

func New(tenant string, appendable storage.Appendable) gen.Processor {
	return &processor{
		namespace:           "tempo",
		tenant:              tenant,
		appendable:          appendable,
		calls:               make(map[string]float64),
		latencyCount:        make(map[string]float64),
		latencySum:          make(map[string]float64),
		latencyBucketCounts: make(map[string][]float64),
		// TODO: make this configurable.
		latencyBuckets: []float64{1, 10, 50, 100, 500},
		cache:          make(map[string]labels.Labels),
	}
}

func (p *processor) Name() string { return name }

func (p *processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	p.aggregateMetrics(req.Batches)

	return p.collectMetrics(ctx)
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

	latencyMS := float64(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano()) / float64(time.Millisecond.Nanoseconds())

	p.mtx.Lock()
	p.cacheLabels(key, svcName, span)
	p.calls[key]++
	p.aggregateLatencyMetrics(key, latencyMS)
	p.mtx.Unlock()
}

func (p *processor) aggregateLatencyMetrics(key string, latencyMS float64) {
	// TODO: make this configurable
	if _, ok := p.latencyBucketCounts[key]; !ok {
		p.latencyBucketCounts[key] = make([]float64, len(p.latencyBuckets)+1)
	}

	p.latencyCount[key]++
	p.latencySum[key] += latencyMS
	idx := sort.SearchFloat64s(p.latencyBuckets, latencyMS)
	for i := 0; i < idx; i++ {
		p.latencyBucketCounts[key][i]++
	}
}

func (p *processor) collectMetrics(ctx context.Context) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	t := time.Now().Unix()

	appender := p.appendable.Appender(ctx)

	if err := p.collectCalls(appender, t); err != nil {
		return err
	}

	if err := p.collectLatencyMetrics(appender, t); err != nil {
		return err
	}

	return appender.Commit()
}

func (p *processor) collectCalls(appender storage.Appender, t int64) error {
	// TODO: only collect new data points.
	for key, count := range p.calls {
		lbls := p.getLabels(key, callsMetric)

		if _, err := appender.Append(0, lbls, t, count); err != nil {
			return err
		}
	}
	return nil
}

func (p *processor) collectLatencyMetrics(appender storage.Appender, t int64) error {
	// TODO: iterate only once.
	for key, count := range p.latencyCount {
		lbls := p.getLabels(key, "latency_count")

		if _, err := appender.Append(0, lbls, t, count); err != nil {
			return err
		}
	}
	for key, count := range p.latencySum {
		lbls := p.getLabels(key, "latency_sum")

		if _, err := appender.Append(0, lbls, t, count); err != nil {
			return err
		}
	}
	for key, buckets := range p.latencyBucketCounts {
		lbls := p.getLabels(key, "latency_bucket")
		for i, count := range buckets {
			if i < len(p.latencyBuckets) {
				lbls = append(lbls, labels.Label{Name: "le", Value: strconv.FormatInt(int64(p.latencyBuckets[i]), 10)})
				if _, err := appender.Append(0, lbls, t, count); err != nil {
					return err
				}
			} else {
				lbls = append(lbls, labels.Label{Name: "le", Value: "+Inf"})
				if _, err := appender.Append(0, lbls, t, count); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *processor) buildKey(svcName string, span *v1_trace.Span) string {
	// TODO: add more dimensions
	key := fmt.Sprintf("%s_%s_%s_%s", svcName, span.Name, span.Kind, span.Status)

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
