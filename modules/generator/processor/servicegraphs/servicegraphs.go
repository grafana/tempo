package servicegraphs

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs/store"
	"github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

var (
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
	cfg Config

	store store.Store

	// TODO do we want to keep this? See other TODO note
	// completed edges are pushed through this channel to be processed.
	//collectCh chan string

	serviceGraphRequestTotal           *prometheus.CounterVec
	serviceGraphRequestFailedTotal     *prometheus.CounterVec
	serviceGraphRequestServerHistogram *prometheus.HistogramVec
	serviceGraphRequestClientHistogram *prometheus.HistogramVec
	serviceGraphUnpairedSpansTotal     *prometheus.CounterVec
	serviceGraphDroppedSpansTotal      *prometheus.CounterVec

	metricDroppedSpans  prometheus.Counter
	metricUnpairedEdges prometheus.Counter
}

func New(cfg Config, tenant string) gen.Processor {
	p := &processor{
		cfg: cfg,

		// TODO I've commented out this code for now since we are not reading this channel anywhere, I believe this is causing a memory leak
		//  completed edges are collected during store.Expire(), we should decided whether this is okay or not
		//collectCh: make(chan string, cfg.MaxItems),

		// TODO we only have to pass tenant to be used in instrumentation, can we avoid doing this somehow?
		metricDroppedSpans:  metricDroppedSpans.WithLabelValues(tenant),
		metricUnpairedEdges: metricUnpairedEdges.WithLabelValues(tenant),
	}

	p.store = store.NewStore(cfg.Wait, cfg.MaxItems, p.collectEdge)

	// TODO quick hack to run store.Expire() in a separate thread, I believe this is causing writes to hang
	//  we should add some logic to clean up this goroutine at shutdown
	go func() {
		ticker := time.NewTicker(2 * time.Second)

		for {
			<-ticker.C
			p.store.Expire()
		}
	}()

	return p
}

func (p *processor) Name() string { return "service_graphs" }

func (p *processor) RegisterMetrics(reg prometheus.Registerer) error {
	p.serviceGraphRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "traces",
		Name:      "service_graph_request_total",
		Help:      "Total count of requests between two nodes",
	}, []string{"client", "server"})
	p.serviceGraphRequestFailedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "traces",
		Name:      "service_graph_request_failed_total",
		Help:      "Total count of failed requests between two nodes",
	}, []string{"client", "server"})
	p.serviceGraphRequestServerHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "traces",
		Name:      "service_graph_request_server_seconds",
		Help:      "Time for a request between two nodes as seen from the server",
		Buckets:   p.cfg.HistogramBuckets,
	}, []string{"client", "server"})
	p.serviceGraphRequestClientHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "traces",
		Name:      "service_graph_request_client_seconds",
		Help:      "Time for a request between two nodes as seen from the client",
		Buckets:   p.cfg.HistogramBuckets,
	}, []string{"client", "server"})
	p.serviceGraphUnpairedSpansTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "traces",
		Name:      "service_graph_unpaired_spans_total",
		Help:      "Total count of unpaired spans",
	}, []string{"client", "server"})
	p.serviceGraphDroppedSpansTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "traces",
		Name:      "service_graph_dropped_spans_total",
		Help:      "Total count of dropped spans",
	}, []string{"client", "server"})

	cs := []prometheus.Collector{
		p.serviceGraphRequestTotal,
		p.serviceGraphRequestFailedTotal,
		p.serviceGraphRequestServerHistogram,
		p.serviceGraphRequestClientHistogram,
		p.serviceGraphUnpairedSpansTotal,
		p.serviceGraphDroppedSpansTotal,
	}

	for _, c := range cs {
		if err := reg.Register(c); err != nil {
			return err
		}
	}

	return nil
}

func (p *processor) UnregisterMetrics(reg prometheus.Registerer) {
	cs := []prometheus.Collector{
		p.serviceGraphRequestTotal,
		p.serviceGraphRequestFailedTotal,
		p.serviceGraphRequestServerHistogram,
		p.serviceGraphRequestClientHistogram,
		p.serviceGraphUnpairedSpansTotal,
		p.serviceGraphDroppedSpansTotal,
	}

	for _, c := range cs {
		reg.Unregister(c)
	}
}

func (p *processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "servicegraphs.PushSpans")
	defer span.Finish()

	if err := p.consume(req.Batches); err != nil {
		if errors.As(err, &tooManySpansError{}) {
			level.Warn(log.Logger).Log("msg", "skipped processing of spans", "maxItems", p.cfg.MaxItems, "err", err)
		} else {
			level.Error(log.Logger).Log("msg", "failed consuming traces", "err", err)
		}
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
				//edge *store.Edge
				k   string
				err error
			)
			for _, span := range ils.Spans {
				switch span.Kind {
				case v1.Span_SPAN_KIND_CLIENT:
					k = key(hex.EncodeToString(span.TraceId), hex.EncodeToString(span.SpanId))
					_, err = p.store.UpsertEdge(k, func(e *store.Edge) {
						e.ClientService = svcName
						e.ClientLatencySec = spanDurationSec(span)
						e.Failed = e.Failed || p.spanFailed(span)
					})
				case v1.Span_SPAN_KIND_SERVER:
					k = key(hex.EncodeToString(span.TraceId), hex.EncodeToString(span.ParentSpanId))
					_, err = p.store.UpsertEdge(k, func(e *store.Edge) {
						e.ServerService = svcName
						e.ServerLatencySec = spanDurationSec(span)
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

				// TODO no one is reading from this channel, we collect completed edges during store.Expire
				//if edge.IsCompleted() {
				//	p.collectCh <- k
				//}
			}
		}
	}

	if totalDroppedSpans > 0 {
		return &tooManySpansError{
			droppedSpans: totalDroppedSpans,
		}
	}

	return nil
}

func (p *processor) Shutdown(ctx context.Context) error {
	return nil
}

// collectEdge records the metrics for the given edge.
// Returns true if the edge is completed or expired and should be deleted.
func (p *processor) collectEdge(e *store.Edge) {
	if e.IsCompleted() {
		p.serviceGraphRequestTotal.WithLabelValues(e.ClientService, e.ServerService).Inc()
		if e.Failed {
			p.serviceGraphRequestFailedTotal.WithLabelValues(e.ClientService, e.ServerService).Inc()
		}
		p.serviceGraphRequestServerHistogram.WithLabelValues(e.ClientService, e.ServerService).Observe(e.ServerLatencySec)
		p.serviceGraphRequestClientHistogram.WithLabelValues(e.ClientService, e.ServerService).Observe(e.ClientLatencySec)
	} else if e.IsExpired() {
		p.metricUnpairedEdges.Inc()
	}
}

func (p *processor) spanFailed(span *v1.Span) bool {
	return false
}

func spanDurationSec(span *v1.Span) float64 {
	return float64(span.EndTimeUnixNano-span.StartTimeUnixNano) / float64(time.Second.Nanoseconds())
}

func key(k1, k2 string) string {
	return fmt.Sprintf("%s-%s", k1, k2)
}
