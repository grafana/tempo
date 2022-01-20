package servicegraphs

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs/store"
	"github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

var (
	// TODO we have this active series metric for every processor - make processor a label and move into a common config?
	//  in fact, can we create a common component that handles active series?
	//  this could also contain common namespace settings
	metricActiveSeries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_active_series",
		Help:      "The amount of series currently active",
	}, []string{"tenant"})
	metricDroppedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_dropped_spans",
		Help:      "Number of dropped spans.",
	}, []string{"tenant"})
	metricUnpairedEdges = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_unpaired_edges",
		Help:      "Number of expired edges (client or server).",
	}, []string{"tenant"})
)

type tooManySpansError struct {
	droppedSpans int
}

func (t tooManySpansError) Error() string {
	return fmt.Sprintf("dropped %d spans", t.droppedSpans)
}

type processor struct {
	cfg       Config
	namespace string

	mtx sync.Mutex

	store store.Store

	wait     time.Duration
	maxItems int

	// completed edges are pushed through this channel to be processed.
	collectCh chan string

	lastUpdate map[string]time.Time
	requests   map[string]float64
	// latency metrics
	clientLatencySum          map[string]float64
	clientLatencyCount        map[string]float64
	clientLatencyBucketCounts map[string][]float64
	serverLatencySum          map[string]float64
	serverLatencyCount        map[string]float64
	serverLatencyBucketCounts map[string][]float64

	latencyBuckets []float64
	cache          map[string]labels.Labels
	dimensions     []string

	metricActiveSeries  prometheus.Gauge
	metricDroppedSpans  prometheus.Counter
	metricUnpairedEdges prometheus.Counter

	// for testing
	now func() time.Time
}

func New(cfg Config, namespace string, tenant string) gen.Processor {
	p := &processor{
		cfg:       cfg,
		namespace: namespace,

		wait:     cfg.Wait,
		maxItems: cfg.MaxItems,

		collectCh: make(chan string, cfg.MaxItems),

		lastUpdate:                make(map[string]time.Time),
		requests:                  make(map[string]float64),
		clientLatencySum:          make(map[string]float64),
		clientLatencyCount:        make(map[string]float64),
		clientLatencyBucketCounts: make(map[string][]float64),
		serverLatencySum:          make(map[string]float64),
		serverLatencyCount:        make(map[string]float64),
		serverLatencyBucketCounts: make(map[string][]float64),
		latencyBuckets:            cfg.HistogramBuckets,
		cache:                     make(map[string]labels.Labels),
		dimensions:                make([]string, 0),

		// TODO we only have to pass tenant to be used in instrumentation, can we avoid doing this somehow?
		metricActiveSeries:  metricActiveSeries.WithLabelValues(tenant),
		metricDroppedSpans:  metricDroppedSpans.WithLabelValues(tenant),
		metricUnpairedEdges: metricUnpairedEdges.WithLabelValues(tenant),

		now: time.Now,
	}

	p.store = store.NewStore(cfg.Wait, cfg.MaxItems, p.collectEdge)

	return p
}

func (p *processor) Name() string { return "service_graphs" }

func (p *processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "servicegraphs.PushSpans")
	defer span.Finish()

	if err := p.consume(req.Batches); err != nil {
		if errors.As(err, &tooManySpansError{}) {
			level.Warn(log.Logger).Log("msg", "skipped processing of spans", "maxItems", p.maxItems, "err", err)
		} else {
			level.Error(log.Logger).Log("msg", "failed consuming traces", "err", err)
		}
		return nil
	}
	return nil
}

func (p *processor) consume(resourceSpans []*v1.ResourceSpans) error {
	var totalDroppedSpans int

	for _, rs := range resourceSpans {
		svcName := util.GetServiceName(rs.Resource)
		if svcName == "" {
			continue
		}

		for _, ils := range rs.InstrumentationLibrarySpans {
			var (
				edge *store.Edge
				k    string
				err  error
			)
			for _, span := range ils.Spans {
				switch span.Kind {
				case v1.Span_SPAN_KIND_CLIENT:
					k = key(hex.EncodeToString(span.TraceId), hex.EncodeToString(span.SpanId))
					edge, err = p.store.UpsertEdge(k, func(e *store.Edge) {
						e.ClientService = svcName
						e.ClientLatency = spanDuration(span)
						e.Failed = e.Failed || p.spanFailed(span)
					})
				case v1.Span_SPAN_KIND_SERVER:
					k = key(hex.EncodeToString(span.TraceId), hex.EncodeToString(span.ParentSpanId))
					edge, err = p.store.UpsertEdge(k, func(e *store.Edge) {
						e.ServerService = svcName
						e.ServerLatency = spanDuration(span)
						e.Failed = e.Failed || p.spanFailed(span)
					})
				default:
					continue
				}

				if errors.Is(err, store.ErrTooManyItems) {
					totalDroppedSpans++
					p.metricDroppedSpans.Inc()
					continue
				}

				// upsertEdge will only return this errTooManyItems
				if err != nil {
					return err
				}

				if edge.IsCompleted() {
					// TODO who is reading from this channel?
					p.collectCh <- k
				}
			}
		}
	}

	// TODO should we return err on dropped spans? This will stop processing by other processors
	//if totalDroppedSpans > 0 {
	//	return &tooManySpansError{
	//		droppedSpans: totalDroppedSpans,
	//	}
	//}

	return nil
}

func (p *processor) CollectMetrics(ctx context.Context, appender storage.Appender) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "servicegraphs.CollectMetrics")
	defer span.Finish()

	// Evict expired edges
	p.store.Expire()

	p.mtx.Lock()
	defer p.mtx.Unlock()

	// remove inactive metrics
	for key, lastUpdate := range p.lastUpdate {
		sinceLastUpdate := p.now().Sub(lastUpdate)
		if sinceLastUpdate > p.cfg.DeleteAfterLastUpdate {
			delete(p.lastUpdate, key)
			delete(p.requests, key)
			delete(p.cache, key)
		}
	}
	p.metricActiveSeries.Set(float64(len(p.requests)))

	timestampMs := p.now().UnixMilli()
	if err := p.collectCounters(appender, timestampMs); err != nil {
		return err
	}
	if err := p.collectHistograms(appender, timestampMs); err != nil {
		return err
	}

	return nil
}

func (p *processor) collectCounters(appender storage.Appender, timestampMs int64) error {
	for key, count := range p.requests {
		lbls := p.getLabels(key, "service_graph_request_total")

		if _, err := appender.Append(0, lbls, timestampMs, count); err != nil {
			return err
		}

		// TODO we should also collect service_graph_request_failed_total
	}
	return nil
}

func (p *processor) collectHistograms(appender storage.Appender, timestampMs int64) error {
	if err := p.collectHistogram(appender, timestampMs, p.clientLatencyCount, p.clientLatencySum, p.clientLatencyBucketCounts, "service_graph_request_client_seconds"); err != nil {
		return err
	}
	if err := p.collectHistogram(appender, timestampMs, p.serverLatencyCount, p.serverLatencySum, p.serverLatencyBucketCounts, "service_graph_request_server_seconds"); err != nil {
		return err
	}
	return nil
}

func (p *processor) collectHistogram(
	appender storage.Appender, timestampMs int64,
	count, sum map[string]float64, bucketCounts map[string][]float64,
	name string,
) error {
	for key := range count {
		// Collect latency count
		lbls := p.getLabels(key, name+"_count")
		if _, err := appender.Append(0, lbls, timestampMs, count[key]); err != nil {
			return err
		}

		// Collect latency sum
		lbls = p.getLabels(key, name+"_sum")
		if _, err := appender.Append(0, lbls, timestampMs, sum[key]); err != nil {
			return err
		}

		// Collect latency buckets
		for i, count := range bucketCounts[key] {
			if i == len(p.latencyBuckets) {
				lbls = append(p.getLabels(key, name+"_bucket"), labels.Label{Name: "le", Value: "+Inf"})
			} else {
				lbls = append(p.getLabels(key, name+"_bucket"), labels.Label{Name: "le", Value: strconv.FormatFloat(p.latencyBuckets[i], 'f', -1, 64)})
			}
			if _, err := appender.Append(0, lbls, timestampMs, count); err != nil {
				return err
			}
		}

	}
	return nil
}

// Must be called under lock
func (p *processor) getLabels(key, metricName string) labels.Labels {
	// TODO: check if it doesn't exist?
	lbls := p.cache[key]

	lbls = append(lbls, labels.Label{Name: "__name__", Value: fmt.Sprintf("%s_%s", p.namespace, metricName)})

	return lbls
}

func (p *processor) Shutdown(ctx context.Context) error {
	return nil
}

// collectEdge records the metrics for the given edge.
// Returns true if the edge is completed or expired and should be deleted.
func (p *processor) collectEdge(e *store.Edge) {
	if e.IsCompleted() {
		key, lbls := p.buildKey(e.ClientService, e.ServerService)

		p.mtx.Lock()
		p.lastUpdate[key] = p.now()
		p.cache[key] = lbls
		p.requests[key]++
		p.aggregateLatencyMetrics(key, e)
		// TODO: record failed metric
		p.mtx.Unlock()

	} else if e.IsExpired() {
		p.metricUnpairedEdges.Inc()
	}
}

func (p *processor) aggregateLatencyMetrics(key string, e *store.Edge) {
	p.aggregateClientLatencyMetrics(key, e.ClientLatency)
	p.aggregateServerLatencyMetrics(key, e.ServerLatency)
}

func (p *processor) aggregateClientLatencyMetrics(key string, latencyMS float64) {
	if _, ok := p.clientLatencyBucketCounts[key]; !ok {
		p.clientLatencyBucketCounts[key] = make([]float64, len(p.latencyBuckets)+1)
	}

	p.clientLatencyCount[key]++
	p.clientLatencySum[key] += latencyMS
	idx := sort.SearchFloat64s(p.latencyBuckets, latencyMS)
	for i := 0; i < idx; i++ {
		p.clientLatencyBucketCounts[key][i]++
	}
}

func (p *processor) aggregateServerLatencyMetrics(key string, latencyMS float64) {
	if _, ok := p.serverLatencyBucketCounts[key]; !ok {
		p.serverLatencyBucketCounts[key] = make([]float64, len(p.latencyBuckets)+1)
	}

	p.serverLatencyCount[key]++
	p.serverLatencySum[key] += latencyMS
	idx := sort.SearchFloat64s(p.latencyBuckets, latencyMS)
	for i := 0; i < idx; i++ {
		p.serverLatencyBucketCounts[key][i]++
	}
}

func (p *processor) buildKey(client, server string) (string, labels.Labels) {
	lbls := labels.Labels{
		labels.Label{Name: "client", Value: client},
		labels.Label{Name: "server", Value: server},
	}
	return fmt.Sprintf("%s_%s", client, server), lbls
}

func (p *processor) spanFailed(span *v1.Span) bool {
	return false
}

func spanDuration(span *v1.Span) float64 {
	return float64(span.EndTimeUnixNano-span.StartTimeUnixNano) / float64(time.Millisecond.Nanoseconds())
}

func key(k1, k2 string) string {
	return fmt.Sprintf("%s-%s", k1, k2)
}
