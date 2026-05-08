package servicegraphs

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs/store"
	processor_util "github.com/grafana/tempo/modules/generator/processor/util"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/validation"
	"github.com/grafana/tempo/pkg/cache/reclaimable"
	"github.com/grafana/tempo/pkg/spanfilter"
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
	metricDroppedEdges = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_dropped_edges_total",
		Help:      "Number of edges dropped due to matching a dropped span side counterpart",
	}, []string{"tenant"})
	metricDroppedSpanSideCacheOverflows = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_processor_service_graphs_dropped_span_side_cache_overflow_total",
		Help:      "Number of dropped span side cache insertions skipped because the cache reached max items",
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
	metricConnectionInfo                = "traces_service_graph_connection_info"
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
	store    store.Store

	closeCh chan struct{}

	serviceGraphRequestTotal                           registry.Counter
	serviceGraphRequestFailedTotal                     registry.Counter
	serviceGraphRequestServerSecondsHistogram          registry.Histogram
	serviceGraphRequestClientSecondsHistogram          registry.Histogram
	serviceGraphRequestMessagingSystemSecondsHistogram registry.Histogram
	serviceGraphConnectionInfo                         registry.Gauge
	dimensionLabels                                    []dimensionLabel
	filter                                             *spanfilter.SpanFilter
	usesSpanMultiplier                                 bool

	filteredSpansCounter                prometheus.Counter
	metricDroppedSpans                  prometheus.Counter
	metricDroppedEdges                  prometheus.Counter
	metricDroppedSpanSideCacheOverflows prometheus.Counter
	metricTotalEdges                    prometheus.Counter
	metricExpiredEdges                  prometheus.Counter
	invalidUTF8Counter                  prometheus.Counter
	logger                              log.Logger
}

type dimensionLabel struct {
	name        string
	label       string
	clientName  string
	clientLabel string
	serverName  string
	serverLabel string
}

func New(cfg Config, tenant string, reg registry.Registry, logger log.Logger, filteredSpansCounter, invalidUTF8Counter prometheus.Counter) (gen.Processor, error) {
	if cfg.EnableVirtualNodeLabel {
		cfg.Dimensions = append(cfg.Dimensions, virtualNodeLabel)
	}

	sanitizeCache := reclaimable.New(validation.SanitizeLabelName, 10000)
	dimensionLabels := make([]dimensionLabel, len(cfg.Dimensions))
	for i, dim := range cfg.Dimensions {
		clientName := "client_" + dim
		serverName := "server_" + dim
		dimensionLabels[i] = dimensionLabel{
			name:        dim,
			label:       sanitizeCache.Get(dim),
			clientName:  clientName,
			clientLabel: sanitizeCache.Get(clientName),
			serverName:  serverName,
			serverLabel: sanitizeCache.Get(serverName),
		}
	}

	filter, err := spanfilter.NewSpanFilter(cfg.FilterPolicies)
	if err != nil {
		return nil, err
	}

	p := &Processor{
		Cfg:      cfg,
		registry: reg,
		closeCh:  make(chan struct{}, 1),

		dimensionLabels:    dimensionLabels,
		filter:             filter,
		usesSpanMultiplier: cfg.SpanMultiplierKey != "" || cfg.EnableTraceStateSpanMultiplier,

		filteredSpansCounter:                filteredSpansCounter,
		metricDroppedSpans:                  metricDroppedSpans.WithLabelValues(tenant),
		metricDroppedEdges:                  metricDroppedEdges.WithLabelValues(tenant),
		metricDroppedSpanSideCacheOverflows: metricDroppedSpanSideCacheOverflows.WithLabelValues(tenant),
		metricTotalEdges:                    metricTotalEdges.WithLabelValues(tenant),
		metricExpiredEdges:                  metricExpiredEdges.WithLabelValues(tenant),
		invalidUTF8Counter:                  invalidUTF8Counter,
		logger:                              log.With(logger, "component", "service-graphs"),
	}

	if cfg.Subprocessors[Request] {
		p.serviceGraphRequestTotal = reg.NewCounter(metricRequestTotal)
		p.serviceGraphRequestFailedTotal = reg.NewCounter(metricRequestFailedTotal)
	}
	if cfg.Subprocessors[Latency] {
		p.serviceGraphRequestServerSecondsHistogram = reg.NewHistogram(metricRequestServerSeconds, cfg.HistogramBuckets, cfg.HistogramOverride)
		p.serviceGraphRequestClientSecondsHistogram = reg.NewHistogram(metricRequestClientSeconds, cfg.HistogramBuckets, cfg.HistogramOverride)
		if cfg.EnableMessagingSystemLatencyHistogram {
			p.serviceGraphRequestMessagingSystemSecondsHistogram = reg.NewHistogram(metricRequestMessagingSystemSeconds, cfg.HistogramBuckets, cfg.HistogramOverride)
		}
	}
	if cfg.Subprocessors[ConnectionInfo] {
		p.serviceGraphConnectionInfo = reg.NewGauge(metricConnectionInfo)
	}

	p.store = store.NewStore(cfg.Wait, cfg.MaxItems, p.onComplete, p.onExpire, p.metricDroppedSpanSideCacheOverflows)

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

	return p, nil
}

func (p *Processor) Name() string {
	return gen.ServiceGraphsName
}

func (p *Processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) {
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
				// Non-edge spans are ignored by this processor, so skip filter evaluation too.
				if !isClient(span.Kind) && !isServer(span.Kind) {
					continue
				}

				if !p.filter.ApplyFilterPolicy(rs.Resource, span) {
					p.addDroppedSpanSide(span)
					p.filteredSpansCounter.Inc()
					continue
				}

				connectionType := store.Unknown
				spanMultiplier := 1.0
				if p.usesSpanMultiplier {
					spanMultiplier = processor_util.GetSpanMultiplier(p.Cfg.SpanMultiplierKey, span, rs.Resource, p.Cfg.EnableTraceStateSpanMultiplier)
				}
				switch span.Kind {
				case v1_trace.Span_SPAN_KIND_PRODUCER:
					// override connection type and continue processing as span kind client
					connectionType = store.MessagingSystem
					fallthrough
				case v1_trace.Span_SPAN_KIND_CLIENT:
					isNew, err = store.UpsertEdgeFromBytesWith(p.store, span.TraceId, span.SpanId, store.Client, clientEdgeUpdate{
						p:              p,
						resourceAttr:   rs.Resource.Attributes,
						span:           span,
						svcName:        svcName,
						connectionType: connectionType,
						spanMultiplier: spanMultiplier,
					}, updateClientEdge)

				case v1_trace.Span_SPAN_KIND_CONSUMER:
					// override connection type and continue processing as span kind server
					connectionType = store.MessagingSystem
					fallthrough
				case v1_trace.Span_SPAN_KIND_SERVER:
					if len(span.ParentSpanId) == 0 {
						isNew, err = store.UpsertEdgeFromBytesWith(p.store, span.TraceId, span.ParentSpanId, store.Server, serverEdgeUpdate{
							p:              p,
							resourceAttr:   rs.Resource.Attributes,
							span:           span,
							svcName:        svcName,
							connectionType: connectionType,
							spanMultiplier: spanMultiplier,
							root:           true,
						}, updateServerEdge)
					} else {
						isNew, err = store.UpsertEdgeFromBytesWith(p.store, span.TraceId, span.ParentSpanId, store.Server, serverEdgeUpdate{
							p:              p,
							resourceAttr:   rs.Resource.Attributes,
							span:           span,
							svcName:        svcName,
							connectionType: connectionType,
							spanMultiplier: spanMultiplier,
						}, updateServerEdge)
					}
				}

				switch {
				case errors.Is(err, store.ErrTooManyItems):
					totalDroppedSpans++
					p.metricDroppedSpans.Inc()
					continue
				case errors.Is(err, store.ErrDroppedSpanSide):
					p.metricDroppedEdges.Inc()
					continue
				case err != nil:
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

type clientEdgeUpdate struct {
	p              *Processor
	resourceAttr   []*v1_common.KeyValue
	span           *v1_trace.Span
	svcName        string
	connectionType store.ConnectionType
	spanMultiplier float64
}

func updateClientEdge(e *store.Edge, u clientEdgeUpdate) {
	// TraceID is forwarded to histogram exemplars (which keep the string after
	// scrape) and the edge is recycled with its keyBuf preserved, so this must
	// be a freshly owned string, not a substring of any pooled buffer.
	e.TraceID = tempo_util.TraceIDToHexString(u.span.TraceId)
	e.ConnectionType = u.connectionType
	e.ClientService = u.svcName
	e.ClientLatencySec = spanDurationSec(u.span)
	e.ClientEndTimeUnixNano = u.span.EndTimeUnixNano
	e.Failed = e.Failed || u.p.spanFailed(u.span)
	u.p.upsertDimensions("client_", e.Dimensions, u.resourceAttr, u.span.Attributes)
	e.SpanMultiplier = u.spanMultiplier
	u.p.upsertPeerNode(e, u.span.Attributes)
	u.p.upsertDatabaseRequest(e, u.resourceAttr, u.span)
}

type serverEdgeUpdate struct {
	p              *Processor
	resourceAttr   []*v1_common.KeyValue
	span           *v1_trace.Span
	svcName        string
	connectionType store.ConnectionType
	spanMultiplier float64
	root           bool
}

func updateServerEdge(e *store.Edge, u serverEdgeUpdate) {
	if u.root {
		e.TraceID = tempo_util.TraceIDToHexString(u.span.TraceId)
	}
	e.ConnectionType = u.connectionType
	e.ServerService = u.svcName
	e.ServerLatencySec = spanDurationSec(u.span)
	e.ServerStartTimeUnixNano = u.span.StartTimeUnixNano
	e.Failed = e.Failed || u.p.spanFailed(u.span)
	u.p.upsertDimensions("server_", e.Dimensions, u.resourceAttr, u.span.Attributes)
	e.SpanMultiplier = u.spanMultiplier
	if u.root {
		// PeerNode is only used for root server-span virtual-node inference.
		u.p.upsertPeerNode(e, u.span.Attributes)
	}
}

func (p *Processor) upsertDimensions(prefix string, m map[string]string, resourceAttr, spanAttr []*v1_common.KeyValue) {
	for _, dim := range p.dimensionLabels {
		if v, ok := processor_util.FindAttributeValue(dim.name, resourceAttr, spanAttr); ok {
			if p.Cfg.EnableClientServerPrefix {
				if prefix == "client_" {
					m[dim.clientName] = v
				} else {
					m[dim.serverName] = v
				}
			} else {
				m[dim.name] = v
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

	// Check for db.name or db.namespace first.  The dbName is set initially to maintain backwards compatbility.
	for _, attrName := range p.Cfg.DatabaseNameAttributes {
		if name, ok := processor_util.FindAttributeValue(attrName, resourceAttr, span.Attributes); ok {
			dbName = name
			isDatabase = true
			break
		}
	}

	// If neither db.system nor db.name are present, we can't determine if this is a database request
	if !isDatabase {
		return
	}
	e.ConnectionType = store.Database
	e.ServerLatencySec = spanDurationSec(span)

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
			e.ServerService = host + ":" + port
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
	builder := p.registry.NewLabelBuilder()
	builder.Add("client", e.ClientService)
	builder.Add("server", e.ServerService)
	builder.Add("connection_type", string(e.ConnectionType))

	for _, dimension := range p.dimensionLabels {
		if p.Cfg.EnableClientServerPrefix {
			if p.Cfg.EnableVirtualNodeLabel {
				// leave the extra label for this feature as-is
				if dimension.name == virtualNodeLabel {
					builder.Add(virtualNodeLabel, e.Dimensions[dimension.name])
					continue
				}
			}
			builder.Add(dimension.clientLabel, e.Dimensions[dimension.clientName])
			builder.Add(dimension.serverLabel, e.Dimensions[dimension.serverName])
		} else {
			builder.Add(dimension.label, e.Dimensions[dimension.name])
		}
	}

	registryLabelValues, validUTF8 := builder.CloseAndBorrowLabels()
	if !validUTF8 {
		p.invalidUTF8Counter.Inc()
		return
	}
	updateTimeMs := time.Now().UnixMilli()

	if p.Cfg.Subprocessors[Request] {
		p.serviceGraphRequestTotal.IncWithHashAt(registryLabelValues.Labels, registryLabelValues.Hash, 1*e.SpanMultiplier, updateTimeMs)
		if e.Failed {
			p.serviceGraphRequestFailedTotal.IncWithHashAt(registryLabelValues.Labels, registryLabelValues.Hash, 1*e.SpanMultiplier, updateTimeMs)
		}
	}

	if p.Cfg.Subprocessors[Latency] {
		p.serviceGraphRequestServerSecondsHistogram.ObserveWithExemplarWithHashAt(registryLabelValues.Labels, registryLabelValues.Hash, e.ServerLatencySec, e.TraceID, e.SpanMultiplier, updateTimeMs)
		p.serviceGraphRequestClientSecondsHistogram.ObserveWithExemplarWithHashAt(registryLabelValues.Labels, registryLabelValues.Hash, e.ClientLatencySec, e.TraceID, e.SpanMultiplier, updateTimeMs)

		if p.Cfg.EnableMessagingSystemLatencyHistogram && e.ConnectionType == store.MessagingSystem {
			messagingSystemLatencySec := unixNanosDiffSec(e.ClientEndTimeUnixNano, e.ServerStartTimeUnixNano)
			if messagingSystemLatencySec == 0 {
				level.Warn(p.logger).Log("msg", "producerSpanEndTime must be smaller than consumerSpanStartTime. maybe the peers clocks are not synced", "messagingSystemLatencySec", messagingSystemLatencySec, "traceID", e.TraceID)
			} else {
				p.serviceGraphRequestMessagingSystemSecondsHistogram.ObserveWithExemplarWithHashAt(registryLabelValues.Labels, registryLabelValues.Hash, messagingSystemLatencySec, e.TraceID, e.SpanMultiplier, updateTimeMs)
			}
		}
	}

	if p.Cfg.Subprocessors[ConnectionInfo] {
		// Presence-only info metric: held at 1 while the edge is being observed.
		// The series goes stale and is pruned once edges stop completing for this label set.
		p.serviceGraphConnectionInfo.Set(registryLabelValues.Labels, 1)
	}

	registryLabelValues.Release()
}

func (p *Processor) onExpire(e *store.Edge) {
	wasCounted := false

	// If an edge is expired, we check if there are signs that the missing span is belongs to a "virtual node".
	// These are nodes that are outside the user's reach (eg. an external service for payment processing),
	// or that are not instrumented (eg. a frontend application).
	e.ConnectionType = store.VirtualNode
	if len(e.ClientService) == 0 {
		// If the client service is not set, it means that the span could have been initiated by an external system,
		// like a frontend application or an engineer via `curl`.
		// We check if the span we have is the root span, and if so, we set the client service appropriately.
		if _, parentSpan := parseKey(e.Key()); len(parentSpan) == 0 {

			// If a peer attribute is present, it is used to name the external client service.
			if len(e.PeerNode) > 0 {
				e.ClientService = e.PeerNode
			} else {
				// Request came from an unknown source. No information inferred from the peer attributes.
				e.ClientService = "user"
			}

			if p.Cfg.EnableVirtualNodeLabel {
				e.Dimensions[virtualNodeLabel] = "client"
			}

			p.onComplete(e)
			wasCounted = true
		}
	} else if len(e.ServerService) == 0 && len(e.PeerNode) > 0 {
		// If client span does not have its matching server span, but has a peer attribute present,
		// we make the assumption that a call was made to an external service, for which Tempo won't receive spans.
		e.ServerService = e.PeerNode

		if p.Cfg.EnableVirtualNodeLabel {
			e.Dimensions[virtualNodeLabel] = "server"
		}

		p.onComplete(e)
		wasCounted = true
	}

	// there was no match and no information in the one found span to create a service graph edge. mark expired
	if !wasCounted {
		p.metricExpiredEdges.Inc()
	}
}

func (p *Processor) addDroppedSpanSide(span *v1_trace.Span) {
	if isClient(span.Kind) {
		key := buildKeyFromBytes(span.TraceId, span.SpanId)
		if p.store.AddDroppedSpanSide(key, store.Client) {
			p.metricDroppedEdges.Inc()
		}
		return
	}

	if isServer(span.Kind) {
		// Root server spans have no parent span ID and cannot match a client counterpart.
		if len(span.ParentSpanId) == 0 {
			return
		}

		key := buildKeyFromBytes(span.TraceId, span.ParentSpanId)
		if p.store.AddDroppedSpanSide(key, store.Server) {
			p.metricDroppedEdges.Inc()
		}
	}
}

func isClient(kind v1_trace.Span_SpanKind) bool {
	return kind == v1_trace.Span_SPAN_KIND_CLIENT || kind == v1_trace.Span_SPAN_KIND_PRODUCER
}

func isServer(kind v1_trace.Span_SpanKind) bool {
	return kind == v1_trace.Span_SPAN_KIND_SERVER || kind == v1_trace.Span_SPAN_KIND_CONSUMER
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
	return k1 + "-" + k2
}

func buildKeyFromBytes(k1, k2 []byte) string {
	k1Len := hex.EncodedLen(len(k1))
	buf := make([]byte, k1Len+1+hex.EncodedLen(len(k2)))
	hex.Encode(buf[:k1Len], k1)
	buf[k1Len] = '-'
	hex.Encode(buf[k1Len+1:], k2)
	// The buffer is private and is not mutated after conversion.
	return unsafe.String(unsafe.SliceData(buf), len(buf))
}

func parseKey(key string) (string, string) {
	traceID, spanID, _ := strings.Cut(key, "-")
	return traceID, spanID
}
