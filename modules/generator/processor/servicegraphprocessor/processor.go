package servicegraphprocessor

import (
	"context"
	"errors"
	"fmt"
	"time"

	util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
	"google.golang.org/grpc/codes"
)

type tooManySpansError struct {
	droppedSpans int
}

func (t tooManySpansError) Error() string {
	return fmt.Sprintf("dropped %d spans", t.droppedSpans)
}

// edge is an edge between two nodes in the graph
type edge struct {
	key string

	serverService, clientService string
	serverLatency, clientLatency time.Duration

	// If either the client or the server spans have status code error,
	// the edge will be considered as failed.
	failed bool

	// expiration is the time at which the edge expires, expressed as Unix time
	expiration int64
}

func newEdge(key string, ttl time.Duration) *edge {
	return &edge{
		key: key,

		expiration: time.Now().Add(ttl).Unix(),
	}
}

// isCompleted returns true if the corresponding client and server
// pair spans have been processed for the given edge
func (e *edge) isCompleted() bool {
	return len(e.clientService) != 0 && len(e.serverService) != 0
}

func (e *edge) isExpired() bool {
	return time.Now().Unix() >= e.expiration
}

var _ component.TracesProcessor = (*processor)(nil)

type processor struct {
	nextConsumer consumer.Traces
	reg          prometheus.Registerer

	store *store

	wait     time.Duration
	maxItems int

	// completed edges are pushed through this channel to be processed.
	collectCh chan string

	serviceGraphRequestTotal           *prometheus.CounterVec
	serviceGraphRequestFailedTotal     *prometheus.CounterVec
	serviceGraphRequestServerHistogram *prometheus.HistogramVec
	serviceGraphRequestClientHistogram *prometheus.HistogramVec
	serviceGraphUnpairedSpansTotal     *prometheus.CounterVec
	serviceGraphDroppedSpansTotal      *prometheus.CounterVec

	httpSuccessCodeMap map[int]struct{}
	grpcSuccessCodeMap map[int]struct{}

	logger  log.Logger
	closeCh chan struct{}
}

func NewProcessor(nextConsumer consumer.Traces, cfg *Config, reg prometheus.Registerer) *processor {
	logger := log.With(util.Logger, "component", "service graphs")

	if cfg.Wait == 0 {
		cfg.Wait = DefaultWait
	}
	if cfg.MaxItems == 0 {
		cfg.MaxItems = DefaultMaxItems
	}
	if cfg.Workers == 0 {
		cfg.Workers = DefaultWorkers
	}

	var (
		httpSuccessCodeMap = make(map[int]struct{})
		grpcSuccessCodeMap = make(map[int]struct{})
	)
	if cfg.SuccessCodes != nil {
		for _, sc := range cfg.SuccessCodes.http {
			httpSuccessCodeMap[int(sc)] = struct{}{}
		}
		for _, sc := range cfg.SuccessCodes.grpc {
			grpcSuccessCodeMap[int(sc)] = struct{}{}
		}
	}

	p := &processor{
		nextConsumer: nextConsumer,
		reg:          reg,
		logger:       logger,

		wait:               cfg.Wait,
		maxItems:           cfg.MaxItems,
		httpSuccessCodeMap: httpSuccessCodeMap,
		grpcSuccessCodeMap: grpcSuccessCodeMap,

		collectCh: make(chan string, cfg.Workers),

		closeCh: make(chan struct{}, 1),
	}

	for i := 0; i < cfg.Workers; i++ {
		go func() {
			for {
				select {
				case k := <-p.collectCh:
					p.store.evictEdgeWithLock(k)

				case <-p.closeCh:
					return
				}
			}
		}()
	}

	return p
}

func (p *processor) Start(ctx context.Context, _ component.Host) error {
	// initialize store
	p.store = newStore(p.wait, p.maxItems, p.collectEdge)

	//reg, ok := ctx.Value(contextkeys.PrometheusRegisterer).(prometheus.Registerer)
	//if !ok || reg == nil {
	//	return fmt.Errorf("key does not contain a prometheus registerer")
	//}
	//p.reg = reg
	return p.registerMetrics()
}

func (p *processor) registerMetrics() error {
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
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 12),
	}, []string{"client", "server"})
	p.serviceGraphRequestClientHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "traces",
		Name:      "service_graph_request_client_seconds",
		Help:      "Time for a request between two nodes as seen from the client",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 12),
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
		if err := p.reg.Register(c); err != nil {
			return err
		}
	}

	return nil
}

func (p *processor) Shutdown(context.Context) error {
	close(p.closeCh)
	p.unregisterMetrics()
	return nil
}

func (p *processor) unregisterMetrics() {
	cs := []prometheus.Collector{
		p.serviceGraphRequestTotal,
		p.serviceGraphRequestFailedTotal,
		p.serviceGraphRequestServerHistogram,
		p.serviceGraphRequestClientHistogram,
		p.serviceGraphUnpairedSpansTotal,
	}

	for _, c := range cs {
		p.reg.Unregister(c)
	}
}

func (p *processor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (p *processor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	// Evict expired edges
	p.store.expire()

	if err := p.consume(td); err != nil {
		if errors.As(err, &tooManySpansError{}) {
			level.Warn(p.logger).Log("msg", "skipped processing of spans", "maxItems", p.maxItems, "err", err)
		} else {
			level.Error(p.logger).Log("msg", "failed consuming traces", "err", err)
		}
		return nil
	}

	return p.nextConsumer.ConsumeTraces(ctx, td)
}

// collectEdge records the metrics for the given edge.
// Returns true if the edge is completed or expired and should be deleted.
func (p *processor) collectEdge(e *edge) {
	if e.isCompleted() {
		p.serviceGraphRequestTotal.WithLabelValues(e.clientService, e.serverService).Inc()
		if e.failed {
			p.serviceGraphRequestFailedTotal.WithLabelValues(e.clientService, e.serverService).Inc()
		}
		p.serviceGraphRequestServerHistogram.WithLabelValues(e.clientService, e.serverService).Observe(e.serverLatency.Seconds())
		p.serviceGraphRequestClientHistogram.WithLabelValues(e.clientService, e.serverService).Observe(e.clientLatency.Seconds())
	} else if e.isExpired() {
		p.serviceGraphUnpairedSpansTotal.WithLabelValues(e.clientService, e.serverService).Inc()
	}
}

func (p *processor) consume(trace pdata.Traces) error {
	var totalDroppedSpans int
	rSpansSlice := trace.ResourceSpans()

	for i := 0; i < rSpansSlice.Len(); i++ {
		rSpan := rSpansSlice.At(i)

		svc, ok := rSpan.Resource().Attributes().Get(semconv.AttributeServiceName)
		if !ok || svc.StringVal() == "" {
			continue
		}

		ilsSlice := rSpan.InstrumentationLibrarySpans()
		for j := 0; j < ilsSlice.Len(); j++ {
			ils := ilsSlice.At(j)

			for k := 0; k < ils.Spans().Len(); k++ {

				span := ils.Spans().At(k)

				switch span.Kind() {
				case pdata.SpanKindClient:
					k := key(span.TraceID().HexString(), span.SpanID().HexString())

					edge, err := p.store.upsertEdge(k, func(e *edge) {
						e.clientService = svc.StringVal()
						e.clientLatency = spanDuration(span)
						e.failed = e.failed || p.spanFailed(span) // keep request as failed if any span is failed
					})

					if errors.Is(err, errTooManyItems) {
						totalDroppedSpans++
						p.serviceGraphDroppedSpansTotal.WithLabelValues(svc.StringVal(), "").Inc()
						continue
					}
					// upsertEdge will only return this errTooManyItems
					if err != nil {
						return err
					}

					if edge.isCompleted() {
						p.collectCh <- k
					}

				case pdata.SpanKindServer:
					k := key(span.TraceID().HexString(), span.ParentSpanID().HexString())

					edge, err := p.store.upsertEdge(k, func(e *edge) {
						e.serverService = svc.StringVal()
						e.serverLatency = spanDuration(span)
						e.failed = e.failed || p.spanFailed(span) // keep request as failed if any span is failed
					})

					if errors.Is(err, errTooManyItems) {
						totalDroppedSpans++
						p.serviceGraphDroppedSpansTotal.WithLabelValues("", svc.StringVal()).Inc()
						continue
					}
					// upsertEdge will only return this errTooManyItems
					if err != nil {
						return err
					}

					if edge.isCompleted() {
						p.collectCh <- k
					}

				default:
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

func (p *processor) spanFailed(span pdata.Span) bool {
	// Request considered failed if status is not 2XX or added as a successful status code
	if statusCode, ok := span.Attributes().Get("http.status_code"); ok {
		sc := int(statusCode.IntVal())
		if _, ok := p.httpSuccessCodeMap[sc]; !ok && sc/100 != 2 {
			return true
		}
	}

	// Request considered failed if status is not OK or added as a successful status code
	if statusCode, ok := span.Attributes().Get("grpc.status_code"); ok {
		sc := int(statusCode.IntVal())
		if _, ok := p.grpcSuccessCodeMap[sc]; !ok && sc != int(codes.OK) {
			return true
		}
	}

	return span.Status().Code() == pdata.StatusCodeError
}

func spanDuration(span pdata.Span) time.Duration {
	return span.EndTimestamp().AsTime().Sub(span.StartTimestamp().AsTime())
}

func key(k1, k2 string) string {
	return fmt.Sprintf("%s-%s", k1, k2)
}
