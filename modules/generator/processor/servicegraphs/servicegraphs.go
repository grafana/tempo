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
	"github.com/prometheus/prometheus/util/strutil"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs/store"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/pkg/util"
)

var (
	metricDroppedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_dropped_spans",
		Help:      "Number of spans dropped when trying to add edges",
	}, []string{"tenant"})
	metricTotalEdges = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_edges",
		Help:      "Total number of unique edges",
	}, []string{"tenant"})
	metricExpiredEdges = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_expired_edges",
		Help:      "Number of edges that expired before finding its matching span",
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

type Processor struct {
	Cfg Config

	registry registry.Registry
	store    store.Store

	closeCh chan struct{}

	serviceGraphRequestTotal                  registry.Counter
	serviceGraphRequestFailedTotal            registry.Counter
	serviceGraphRequestServerSecondsHistogram registry.Histogram
	serviceGraphRequestClientSecondsHistogram registry.Histogram

	metricDroppedSpans prometheus.Counter
	metricTotalEdges   prometheus.Counter
	metricExpiredEdges prometheus.Counter
	logger             log.Logger
}

func New(cfg Config, tenant string, registry registry.Registry, logger log.Logger) gen.Processor {
	labels := []string{"client", "server", "connection_type"}
	for _, d := range cfg.Dimensions {
		labels = append(labels, strutil.SanitizeLabelName(d))
	}

	p := &Processor{
		Cfg:      cfg,
		registry: registry,

		closeCh: make(chan struct{}, 1),

		serviceGraphRequestTotal:                  registry.NewCounter(metricRequestTotal, labels),
		serviceGraphRequestFailedTotal:            registry.NewCounter(metricRequestFailedTotal, labels),
		serviceGraphRequestServerSecondsHistogram: registry.NewHistogram(metricRequestServerSeconds, labels, cfg.HistogramBuckets),
		serviceGraphRequestClientSecondsHistogram: registry.NewHistogram(metricRequestClientSeconds, labels, cfg.HistogramBuckets),

		metricDroppedSpans: metricDroppedSpans.WithLabelValues(tenant),
		metricTotalEdges:   metricTotalEdges.WithLabelValues(tenant),
		metricExpiredEdges: metricExpiredEdges.WithLabelValues(tenant),
		logger:             log.With(logger, "component", "service-graphs"),
	}

	p.store = store.NewStore(cfg.Wait, cfg.MaxItems, p.onComplete, p.onExpire)

	expirationTicker := time.NewTicker(2 * time.Second)
	for i := 0; i < cfg.Workers; i++ {
		go func() {
			for {
				select {
				// Periodically clean expired edges from the store
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

func (p *Processor) Name() string {
	return Name
}

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	span, _ := opentracing.StartSpanFromContext(ctx, "servicegraphs.PushSpans")
	defer span.Finish()

	if err := p.consume(req.Batches); err != nil {
		if errors.As(err, &tooManySpansError{}) {
			level.Warn(p.logger).Log("msg", "skipped processing of spans", "maxItems", p.Cfg.MaxItems, "err", err)
		} else {
			level.Error(p.logger).Log("msg", "failed consuming traces", "err", err)
		}
	}
}

func (p *Processor) consume(resourceSpans []*v1_trace.ResourceSpans) (err error) {
	var (
		isNew             bool
		totalDroppedSpans int
	)

	for _, rs := range resourceSpans {
		svcName, ok := processor_util.FindServiceName(rs.Resource.Attributes)
		if !ok {
			continue
		}

		for _, ils := range rs.ScopeSpans {
			for _, span := range ils.Spans {
				connectionType := store.Unknown
				spanMultiplier := processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span)
				switch span.Kind {
				case v1_trace.Span_SPAN_KIND_PRODUCER:
					// override connection type and continue processing as span kind client
					connectionType = store.MessagingSystem
					fallthrough
				case v1_trace.Span_SPAN_KIND_CLIENT:
					key := buildKey(hex.EncodeToString(span.TraceId), hex.EncodeToString(span.SpanId))
					isNew, err = p.store.UpsertEdge(key, func(e *store.Edge) {
						e.TraceID = tempo_util.TraceIDToHexString(span.TraceId)
						e.ConnectionType = connectionType
						e.ClientService = svcName
						e.ClientLatencySec = spanDurationSec(span)
						e.Failed = e.Failed || p.spanFailed(span)
						p.upsertDimensions(e.Dimensions, rs.Resource.Attributes, span.Attributes)
						e.SpanMultiplier = spanMultiplier

						// A database request will only have one span, we don't wait for the server
						// span but just copy details from the client span
						if dbName, ok := processor_util.FindAttributeValue("db.name", rs.Resource.Attributes, span.Attributes); ok {
							e.ConnectionType = store.Database
							e.ServerService = dbName
							e.ServerLatencySec = spanDurationSec(span)
						}
					})

				case v1_trace.Span_SPAN_KIND_CONSUMER:
					// override connection type and continue processing as span kind server
					connectionType = store.MessagingSystem
					fallthrough
				case v1_trace.Span_SPAN_KIND_SERVER:
					key := buildKey(hex.EncodeToString(span.TraceId), hex.EncodeToString(span.ParentSpanId))
					isNew, err = p.store.UpsertEdge(key, func(e *store.Edge) {
						e.TraceID = tempo_util.TraceIDToHexString(span.TraceId)
						e.ConnectionType = connectionType
						e.ServerService = svcName
						e.ServerLatencySec = spanDurationSec(span)
						e.Failed = e.Failed || p.spanFailed(span)
						p.upsertDimensions(e.Dimensions, rs.Resource.Attributes, span.Attributes)
						e.SpanMultiplier = spanMultiplier
					})
				default:
					// this span is not part of an edge
					continue
				}

				if errors.Is(err, store.ErrTooManyItems) {
					totalDroppedSpans++
					p.metricDroppedSpans.Inc()
					continue
				}

				// UpsertEdge will only return ErrTooManyItems
				if err != nil {
					return err
				}

				if isNew {
					p.metricTotalEdges.Inc()
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

func (p *Processor) upsertDimensions(m map[string]string, resourceAttr []*v1_common.KeyValue, spanAttr []*v1_common.KeyValue) {
	for _, dim := range p.Cfg.Dimensions {
		if v, ok := processor_util.FindAttributeValue(dim, resourceAttr, spanAttr); ok {
			m[dim] = v
		}
	}
}

func (p *Processor) Shutdown(_ context.Context) {
	close(p.closeCh)
}

func (p *Processor) onComplete(e *store.Edge) {
	labelValues := make([]string, 0, 2+len(p.Cfg.Dimensions))
	labelValues = append(labelValues, e.ClientService, e.ServerService, string(e.ConnectionType))

	for _, dimension := range p.Cfg.Dimensions {
		labelValues = append(labelValues, e.Dimensions[dimension])
	}

	registryLabelValues := p.registry.NewLabelValues(labelValues)

	p.serviceGraphRequestTotal.Inc(registryLabelValues, 1*e.SpanMultiplier)
	if e.Failed {
		p.serviceGraphRequestFailedTotal.Inc(registryLabelValues, 1*e.SpanMultiplier)
	}

	p.serviceGraphRequestServerSecondsHistogram.ObserveWithExemplar(registryLabelValues, e.ServerLatencySec, e.TraceID, e.SpanMultiplier)
	p.serviceGraphRequestClientSecondsHistogram.ObserveWithExemplar(registryLabelValues, e.ClientLatencySec, e.TraceID, e.SpanMultiplier)
}

func (p *Processor) onExpire(e *store.Edge) {
	p.metricExpiredEdges.Inc()
}

func (p *Processor) spanFailed(span *v1_trace.Span) bool {
	return span.GetStatus().GetCode() == v1_trace.Status_STATUS_CODE_ERROR
}

func spanDurationSec(span *v1_trace.Span) float64 {
	return float64(span.EndTimeUnixNano-span.StartTimeUnixNano) / float64(time.Second.Nanoseconds())
}

func buildKey(k1, k2 string) string {
	return fmt.Sprintf("%s-%s", k1, k2)
}
