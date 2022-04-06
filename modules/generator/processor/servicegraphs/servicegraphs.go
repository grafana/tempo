package servicegraphs

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs/store"
	"github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
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

const (
	metricRequestTotal         = "traces_service_graph_request_total"
	metricRequestFailedTotal   = "traces_service_graph_request_failed_total"
	metricRequestServerSeconds = "traces_service_graph_request_server_seconds"
	metricRequestClientSeconds = "traces_service_graph_request_client_seconds"
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

	// completed edges are pushed through this channel to be processed.
	collectCh chan string
	closeCh   chan struct{}

	serviceGraphRequestTotal                  registry.Counter
	serviceGraphRequestFailedTotal            registry.Counter
	serviceGraphRequestServerSecondsHistogram registry.Histogram
	serviceGraphRequestClientSecondsHistogram registry.Histogram

	metricDroppedSpans  prometheus.Counter
	metricUnpairedEdges prometheus.Counter
	logger              log.Logger
}

func New(cfg Config, tenant string, registry registry.Registry, logger log.Logger) gen.Processor {
	clientServerLabels := []string{"client", "server"}

	p := &processor{
		cfg: cfg,

		collectCh: make(chan string, cfg.MaxItems),
		closeCh:   make(chan struct{}, 1),

		serviceGraphRequestTotal:                  registry.NewCounter(metricRequestTotal, clientServerLabels),
		serviceGraphRequestFailedTotal:            registry.NewCounter(metricRequestFailedTotal, clientServerLabels),
		serviceGraphRequestServerSecondsHistogram: registry.NewHistogram(metricRequestServerSeconds, clientServerLabels, cfg.HistogramBuckets),
		serviceGraphRequestClientSecondsHistogram: registry.NewHistogram(metricRequestClientSeconds, clientServerLabels, cfg.HistogramBuckets),

		metricDroppedSpans:  metricDroppedSpans.WithLabelValues(tenant),
		metricUnpairedEdges: metricUnpairedEdges.WithLabelValues(tenant),
		logger:              logger,
	}

	p.store = store.NewStore(cfg.Wait, cfg.MaxItems, p.collectEdge)

	expirationTicker := time.NewTicker(2 * time.Second)
	for i := 0; i < cfg.Workers; i++ {
		go func() {
			for {
				select {
				case k := <-p.collectCh:
					p.store.EvictEdge(k)

				// Periodically cleans expired edges from the store
				case <-expirationTicker.C:
					p.store.Expire()

				case <-p.closeCh:
					return
				}
			}
		}()
	}

	return p
}

func (p *processor) Name() string { return Name }

func (p *processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	span, _ := opentracing.StartSpanFromContext(ctx, "servicegraphs.PushSpans")
	defer span.Finish()

	if err := p.consume(req.Batches); err != nil {
		if errors.As(err, &tooManySpansError{}) {
			level.Warn(p.logger).Log("msg", "skipped processing of spans", "maxItems", p.cfg.MaxItems, "err", err)
		} else {
			level.Error(p.logger).Log("msg", "failed consuming traces", "err", err)
		}
	}
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
						e.TraceID = tempo_util.TraceIDToHexString(span.TraceId)
						e.ClientService = svcName
						e.ClientLatencySec = spanDurationSec(span)
						e.Failed = e.Failed || p.spanFailed(span)
					})
				case v1.Span_SPAN_KIND_SERVER:
					k = key(hex.EncodeToString(span.TraceId), hex.EncodeToString(span.ParentSpanId))
					edge, err = p.store.UpsertEdge(k, func(e *store.Edge) {
						e.TraceID = tempo_util.TraceIDToHexString(span.TraceId)
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

				if edge.IsCompleted() {
					p.collectCh <- k
				}
			}
		}
	}

	if totalDroppedSpans > 0 {
		return tooManySpansError{
			droppedSpans: totalDroppedSpans,
		}
	}

	return nil
}

func (p *processor) Shutdown(ctx context.Context) {
	close(p.closeCh)
}

// collectEdge records the metrics for the given edge.
// Returns true if the edge is completed or expired and should be deleted.
func (p *processor) collectEdge(e *store.Edge) {
	if e.IsCompleted() {
		clientServerLabelValues := registry.NewLabelValues([]string{e.ClientService, e.ServerService})

		p.serviceGraphRequestTotal.Inc(clientServerLabelValues, 1)
		if e.Failed {
			p.serviceGraphRequestFailedTotal.Inc(clientServerLabelValues, 1)
		}

		p.serviceGraphRequestServerSecondsHistogram.ObserveWithExemplar(clientServerLabelValues, e.ServerLatencySec, e.TraceID)
		p.serviceGraphRequestClientSecondsHistogram.ObserveWithExemplar(clientServerLabelValues, e.ClientLatencySec, e.TraceID)
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
