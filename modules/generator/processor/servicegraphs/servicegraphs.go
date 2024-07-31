package servicegraphs

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/util/strutil"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	gen "github.com/grafana/tempo/v2/modules/generator/processor"
	"github.com/grafana/tempo/v2/modules/generator/processor/servicegraphs/store"
	processor_util "github.com/grafana/tempo/v2/modules/generator/processor/util"
	"github.com/grafana/tempo/v2/modules/generator/registry"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v2/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/v2/pkg/tempopb/trace/v1"
	tempo_util "github.com/grafana/tempo/v2/pkg/util"
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
	metricRequestTotal                  = "traces_service_graph_request_total"
	metricRequestFailedTotal            = "traces_service_graph_request_failed_total"
	metricRequestServerSeconds          = "traces_service_graph_request_server_seconds"
	metricRequestClientSeconds          = "traces_service_graph_request_client_seconds"
	metricRequestMessagingSystemSeconds = "traces_service_graph_request_messaging_system_seconds"
)

const virtualNodeLabel = "virtual_node"

var defaultPeerAttributes = []attribute.Key{
	semconv.PeerServiceKey, semconv.DBNameKey, semconv.DBSystemKey,
}

type tooManySpansError struct {
	droppedSpans int
}

func (t *tooManySpansError) Error() string {
	return fmt.Sprintf("dropped %d spans", t.droppedSpans)
}

type Processor struct {
	Cfg Config

	registry registry.Registry
	labels   []string
	store    store.Store

	closeCh chan struct{}

	serviceGraphRequestTotal                           registry.Counter
	serviceGraphRequestFailedTotal                     registry.Counter
	serviceGraphRequestServerSecondsHistogram          registry.Histogram
	serviceGraphRequestClientSecondsHistogram          registry.Histogram
	serviceGraphRequestMessagingSystemSecondsHistogram registry.Histogram

	metricDroppedSpans prometheus.Counter
	metricTotalEdges   prometheus.Counter
	metricExpiredEdges prometheus.Counter
	logger             log.Logger
}

func New(cfg Config, tenant string, registry registry.Registry, logger log.Logger) gen.Processor {
	labels := []string{"client", "server", "connection_type"}

	if cfg.EnableVirtualNodeLabel {
		cfg.Dimensions = append(cfg.Dimensions, virtualNodeLabel)
	}

	for _, d := range cfg.Dimensions {
		if cfg.EnableClientServerPrefix {
			if cfg.EnableVirtualNodeLabel {
				// leave the extra label for this feature as-is
				if d == virtualNodeLabel {
					labels = append(labels, strutil.SanitizeLabelName(d))
					continue
				}
			}
			labels = append(labels, strutil.SanitizeLabelName("client_"+d), strutil.SanitizeLabelName("server_"+d))
		} else {
			labels = append(labels, strutil.SanitizeLabelName(d))
		}
	}

	p := &Processor{
		Cfg:      cfg,
		registry: registry,
		labels:   labels,
		closeCh:  make(chan struct{}, 1),

		serviceGraphRequestTotal:                           registry.NewCounter(metricRequestTotal),
		serviceGraphRequestFailedTotal:                     registry.NewCounter(metricRequestFailedTotal),
		serviceGraphRequestServerSecondsHistogram:          registry.NewHistogram(metricRequestServerSeconds, cfg.HistogramBuckets),
		serviceGraphRequestClientSecondsHistogram:          registry.NewHistogram(metricRequestClientSeconds, cfg.HistogramBuckets),
		serviceGraphRequestMessagingSystemSecondsHistogram: registry.NewHistogram(metricRequestMessagingSystemSeconds, cfg.HistogramBuckets),

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
		var tmsErr *tooManySpansError
		if errors.As(err, &tmsErr) {
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
						e.ClientEndTimeUnixNano = span.EndTimeUnixNano
						e.Failed = e.Failed || p.spanFailed(span)
						p.upsertDimensions("client_", e.Dimensions, rs.Resource.Attributes, span.Attributes)
						e.SpanMultiplier = spanMultiplier
						p.upsertPeerNode(e, span.Attributes)
						p.upsertDatabaseRequest(e, rs.Resource.Attributes, span)
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
						e.ServerStartTimeUnixNano = span.StartTimeUnixNano
						e.Failed = e.Failed || p.spanFailed(span)
						p.upsertDimensions("server_", e.Dimensions, rs.Resource.Attributes, span.Attributes)
						e.SpanMultiplier = spanMultiplier
						p.upsertPeerNode(e, span.Attributes)
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
		return &tooManySpansError{
			droppedSpans: totalDroppedSpans,
		}
	}

	return nil
}

func (p *Processor) upsertDimensions(prefix string, m map[string]string, resourceAttr, spanAttr []*v1_common.KeyValue) {
	for _, dim := range p.Cfg.Dimensions {
		if v, ok := processor_util.FindAttributeValue(dim, resourceAttr, spanAttr); ok {
			if p.Cfg.EnableClientServerPrefix {
				m[prefix+dim] = v
			} else {
				m[dim] = v
			}
		}
	}
}

func (p *Processor) upsertPeerNode(e *store.Edge, spanAttr []*v1_common.KeyValue) {
	for _, peerKey := range p.Cfg.PeerAttributes {
		if v, ok := processor_util.FindAttributeValue(peerKey, spanAttr); ok {
			e.PeerNode = v
			return
		}
	}
}

// upsertDatabaseRequest handles the logic of adding a database edge on the
// graph.  If we have a db.name or db.system attribute, we assume this is a
// database request.  The name of the edge is determined by the following
// order:
//
//	if we have a peer.service, use it as the database ServerService
//	if we have a server.address, use it as the database ServerService
//	if we have a network.peer.address, use it as the database ServerService.  Include :port if network.peer.port is present
//	if we have a db.name, use it as the database ServerService, which is the backwards-compatible behavior
func (p *Processor) upsertDatabaseRequest(e *store.Edge, resourceAttr []*v1_common.KeyValue, span *v1_trace.Span) {
	var (
		isDatabase bool

		// The fallback database name
		dbName string
	)

	// Check for db.name first.  The dbName is set initially to maintain backwards compatbility.
	if name, ok := processor_util.FindAttributeValue(string(semconv.DBNameKey), resourceAttr, span.Attributes); ok {
		dbName = name
		isDatabase = true
	}

	// Check for db.system only if we don't have db.name above
	if !isDatabase {
		if _, ok := processor_util.FindAttributeValue(string(semconv.DBSystemKey), resourceAttr, span.Attributes); ok {
			isDatabase = true
		}
	}

	// If neither db.system nor db.name are present, we can't determine if this is a database request
	if !isDatabase {
		return
	}
	e.ConnectionType = store.Database
	e.ServerLatencySec = spanDurationSec(span)

	// Set the service name by order of precedence

	// Check for peer.service
	if name, ok := processor_util.FindAttributeValue(string(semconv.PeerServiceKey), resourceAttr, span.Attributes); ok {
		e.ServerService = name
		return
	}

	// Check for server.address
	if name, ok := processor_util.FindAttributeValue(string(semconv.ServerAddressKey), resourceAttr, span.Attributes); ok {
		e.ServerService = name
		return
	}

	// Check for network.peer.address and network.peer.port.  Use port if it is present.
	if host, ok := processor_util.FindAttributeValue(string(semconv.NetworkPeerAddressKey), resourceAttr, span.Attributes); ok {
		if port, ok := processor_util.FindAttributeValue(string(semconv.NetworkPeerPortKey), resourceAttr, span.Attributes); ok {
			e.ServerService = fmt.Sprintf("%s:%s", host, port)
			return
		}
		e.ServerService = host
		return
	}

	// Fallback to db.name
	if dbName != "" {
		e.ServerService = dbName
	}
}

func (p *Processor) Shutdown(_ context.Context) {
	close(p.closeCh)
}

func (p *Processor) onComplete(e *store.Edge) {
	labelValues := make([]string, 0, 2+len(p.Cfg.Dimensions))
	labelValues = append(labelValues, e.ClientService, e.ServerService, string(e.ConnectionType))

	for _, dimension := range p.Cfg.Dimensions {
		if p.Cfg.EnableClientServerPrefix {
			if p.Cfg.EnableVirtualNodeLabel {
				// leave the extra label for this feature as-is
				if dimension == virtualNodeLabel {
					labelValues = append(labelValues, e.Dimensions[dimension])
					continue
				}
			}
			labelValues = append(labelValues, e.Dimensions["client_"+dimension], e.Dimensions["server_"+dimension])
		} else {
			labelValues = append(labelValues, e.Dimensions[dimension])
		}
	}

	labels := append([]string{}, p.labels...)

	registryLabelValues := p.registry.NewLabelValueCombo(labels, labelValues)

	p.serviceGraphRequestTotal.Inc(registryLabelValues, 1*e.SpanMultiplier)
	if e.Failed {
		p.serviceGraphRequestFailedTotal.Inc(registryLabelValues, 1*e.SpanMultiplier)
	}

	p.serviceGraphRequestServerSecondsHistogram.ObserveWithExemplar(registryLabelValues, e.ServerLatencySec, e.TraceID, e.SpanMultiplier)
	p.serviceGraphRequestClientSecondsHistogram.ObserveWithExemplar(registryLabelValues, e.ClientLatencySec, e.TraceID, e.SpanMultiplier)

	if p.Cfg.EnableMessagingSystemLatencyHistogram && e.ConnectionType == store.MessagingSystem {
		messagingSystemLatencySec := unixNanosDiffSec(e.ClientEndTimeUnixNano, e.ServerStartTimeUnixNano)
		if messagingSystemLatencySec == 0 {
			level.Warn(p.logger).Log("msg", "producerSpanEndTime must be smaller than consumerSpanStartTime. maybe the peers clocks are not synced", "messagingSystemLatencySec", messagingSystemLatencySec, "traceID", e.TraceID)
		} else {
			p.serviceGraphRequestMessagingSystemSecondsHistogram.ObserveWithExemplar(registryLabelValues, messagingSystemLatencySec, e.TraceID, e.SpanMultiplier)
		}
	}
}

func (p *Processor) onExpire(e *store.Edge) {
	p.metricExpiredEdges.Inc()

	// If an edge is expired, we check if there are signs that the missing span is belongs to a "virtual node".
	// These are nodes that are outside the user's reach (eg. an external service for payment processing),
	// or that are not instrumented (eg. a frontend application).
	e.ConnectionType = store.VirtualNode
	if len(e.ClientService) == 0 {
		// If the client service is not set, it means that the span could have been initiated by an external system,
		// like a frontend application or an engineer via `curl`.
		// We check if the span we have is the root span, and if so, we set the client service to "user".
		if _, parentSpan := parseKey(e.Key()); len(parentSpan) == 0 {
			e.ClientService = "user"

			if p.Cfg.EnableVirtualNodeLabel {
				e.Dimensions[virtualNodeLabel] = "client"
			}

			p.onComplete(e)
		}
	} else if len(e.ServerService) == 0 && len(e.PeerNode) > 0 {
		// If client span does not have its matching server span, but has a peer attribute present,
		// we make the assumption that a call was made to an external service, for which Tempo won't receive spans.
		e.ServerService = e.PeerNode

		if p.Cfg.EnableVirtualNodeLabel {
			e.Dimensions[virtualNodeLabel] = "server"
		}

		p.onComplete(e)
	}
}

func (p *Processor) spanFailed(span *v1_trace.Span) bool {
	return span.GetStatus().GetCode() == v1_trace.Status_STATUS_CODE_ERROR
}

func unixNanosDiffSec(unixNanoStart uint64, unixNanoEnd uint64) float64 {
	if unixNanoStart > unixNanoEnd {
		// To prevent underflow, return 0.
		return 0
	}
	// Safe subtraction.
	return float64(unixNanoEnd-unixNanoStart) / float64(time.Second)
}

func spanDurationSec(span *v1_trace.Span) float64 {
	return unixNanosDiffSec(span.StartTimeUnixNano, span.EndTimeUnixNano)
}

func buildKey(k1, k2 string) string {
	return fmt.Sprintf("%s-%s", k1, k2)
}

func parseKey(key string) (string, string) {
	parts := strings.Split(key, "-")
	return parts[0], parts[1]
}
