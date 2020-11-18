package querier

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"

	"github.com/cortexproject/cortex/pkg/ring"
	ring_client "github.com/cortexproject/cortex/pkg/ring/client"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/services"

	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	tempo_util "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/validation"
)

var (
	metricQueryReads = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "query_reads",
		Help:      "count of reads",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	}, []string{"layer"})
	metricQueryBytesRead = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "query_bytes_read",
		Help:      "bytes read",
		Buckets:   prometheus.ExponentialBuckets(1024, 2, 10),
	}, []string{"layer"})
	metricIngesterClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "querier_ingester_clients",
		Help:      "The current number of ingester clients.",
	})
)

// Querier handlers queries.
type Querier struct {
	services.Service

	cfg    Config
	ring   ring.ReadRing
	pool   *ring_client.Pool
	store  storage.Store
	limits *overrides.Overrides

	subservicesWatcher *services.FailureWatcher
}

type responseFromIngesters struct {
	addr     string
	response interface{}
}

// New makes a new Querier.
func New(cfg Config, clientCfg ingester_client.Config, ring ring.ReadRing, store storage.Store, limits *overrides.Overrides) (*Querier, error) {
	factory := func(addr string) (ring_client.PoolClient, error) {
		return ingester_client.New(addr, clientCfg)
	}

	q := &Querier{
		cfg:  cfg,
		ring: ring,
		pool: ring_client.NewPool("querier_pool",
			clientCfg.PoolConfig,
			ring_client.NewRingServiceDiscovery(ring),
			factory,
			metricIngesterClients,
			util.Logger),
		store:  store,
		limits: limits,
	}

	q.subservicesWatcher = services.NewFailureWatcher()
	q.subservicesWatcher.WatchService(q.pool)

	q.Service = services.NewBasicService(q.starting, q.running, q.stopping)
	return q, nil
}

func (q *Querier) starting(ctx context.Context) error {
	err := services.StartAndAwaitRunning(ctx, q.pool)
	if err != nil {
		return fmt.Errorf("failed to start pool %w", err)
	}

	return nil
}

func (q *Querier) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-q.subservicesWatcher.Chan():
		return fmt.Errorf("querier subservices failed %w", err)
	}
}

// Called after distributor is asked to stop via StopAsync.
func (q *Querier) stopping(_ error) error {
	return services.StopAndAwaitTerminated(context.Background(), q.pool)
}

// FindTraceByID implements tempopb.Querier.
func (q *Querier) FindTraceByID(ctx context.Context, req *tempopb.TraceByIDRequest) (*tempopb.TraceByIDResponse, error) {
	if !validation.ValidTraceID(req.TraceID) {
		return nil, fmt.Errorf("invalid trace id")
	}

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.FindTraceByID")
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.FindTraceByID")
	defer span.Finish()

	key := tempo_util.TokenFor(userID, req.TraceID)

	const maxExpectedReplicationSet = 3 // 3.  b/c frigg it
	var descs [maxExpectedReplicationSet]ring.IngesterDesc
	replicationSet, err := q.ring.Get(key, ring.Read, descs[:0])
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.FindTraceByID")
	}

	span.LogFields(ot_log.String("msg", "searching ingesters"))
	// get responses from all ingesters in parallel
	responses, err := q.forGivenIngesters(ctx, replicationSet, func(client tempopb.QuerierClient) (interface{}, error) {
		return client.FindTraceByID(opentracing.ContextWithSpan(ctx, span), req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.FindTraceByID")
	}

	var completeTrace *tempopb.Trace
	for _, r := range responses {
		trace := r.response.(*tempopb.TraceByIDResponse).Trace
		if trace != nil {
			var spanCountA, spanCountB, spanCountTotal int
			completeTrace, spanCountA, spanCountB, spanCountTotal = tempo_util.CombineTraceProtos(completeTrace, trace)
			span.LogFields(ot_log.String("msg", "combined trace protos"),
				ot_log.Int("spansCountA", spanCountA),
				ot_log.Int("spansCountB", spanCountB),
				ot_log.Int("spansCountTotal", spanCountTotal))
		}
	}

	// if the ingester didn't have it check the store.
	if completeTrace == nil {
		foundBytes, metrics, err := q.store.Find(opentracing.ContextWithSpan(ctx, span), userID, req.TraceID)
		if err != nil {
			return nil, errors.Wrap(err, "error querying store in Querier.FindTraceByID")
		}

		out := &tempopb.Trace{}
		err = proto.Unmarshal(foundBytes, out)
		if err != nil {
			return nil, err
		}

		span.LogFields(ot_log.String("msg", "found backend trace"), ot_log.Int("len", len(foundBytes)))
		completeTrace = out
		metricQueryReads.WithLabelValues("bloom").Observe(float64(metrics.BloomFilterReads.Load()))
		metricQueryBytesRead.WithLabelValues("bloom").Observe(float64(metrics.BloomFilterBytesRead.Load()))
		metricQueryReads.WithLabelValues("index").Observe(float64(metrics.IndexReads.Load()))
		metricQueryBytesRead.WithLabelValues("index").Observe(float64(metrics.IndexBytesRead.Load()))
		metricQueryReads.WithLabelValues("block").Observe(float64(metrics.BlockReads.Load()))
		metricQueryBytesRead.WithLabelValues("block").Observe(float64(metrics.BlockBytesRead.Load()))
	}

	return &tempopb.TraceByIDResponse{
		Trace: completeTrace,
	}, nil
}

// forGivenIngesters runs f, in parallel, for given ingesters
func (q *Querier) forGivenIngesters(ctx context.Context, replicationSet ring.ReplicationSet, f func(tempopb.QuerierClient) (interface{}, error)) ([]responseFromIngesters, error) {
	results, err := replicationSet.Do(ctx, q.cfg.ExtraQueryDelay, func(ingester *ring.IngesterDesc) (interface{}, error) {
		client, err := q.pool.GetClientFor(ingester.Addr)
		if err != nil {
			return nil, err
		}

		resp, err := f(client.(tempopb.QuerierClient))
		if err != nil {
			return nil, err
		}

		return responseFromIngesters{ingester.Addr, resp}, nil
	})
	if err != nil {
		return nil, err
	}

	responses := make([]responseFromIngesters, 0, len(results))
	for _, result := range results {
		responses = append(responses, result.(responseFromIngesters))
	}

	return responses, err
}
