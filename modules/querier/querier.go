package querier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	httpgrpc_server "github.com/grafana/dskit/httpgrpc/server"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"

	generator_client "github.com/grafana/tempo/modules/generator/client"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier/worker"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/traceqlmetrics"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var tracer = otel.Tracer("modules/querier")

var (
	metricIngesterClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "querier_ingester_clients",
		Help:      "The current number of ingester clients.",
	})
	metricMetricsGeneratorClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "querier_metrics_generator_clients",
		Help:      "The current number of generator clients.",
	})
)

type (
	forEachFn          func(ctx context.Context, client tempopb.QuerierClient) (any, error)
	forEachGeneratorFn func(ctx context.Context, client tempopb.MetricsGeneratorClient) (any, error)
	replicationSetFn   func(r ring.ReadRing) (ring.ReplicationSet, error)
)

// Querier handlers queries.
type Querier struct {
	services.Service

	cfg Config

	ingesterPools []*ring_client.Pool
	ingesterRings []ring.ReadRing

	generatorPool *ring_client.Pool
	generatorRing ring.ReadRing

	engine *traceql.Engine
	store  storage.Store
	limits overrides.Interface

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New makes a new Querier.
func New(
	cfg Config,
	ingesterClientConfig ingester_client.Config,
	ingesterRings []ring.ReadRing,
	generatorClientConfig generator_client.Config,
	generatorRing ring.ReadRing,
	store storage.Store,
	limits overrides.Interface,
) (*Querier, error) {
	var ingesterClientFactory ring_client.PoolAddrFunc = func(addr string) (ring_client.PoolClient, error) {
		return ingester_client.New(addr, ingesterClientConfig)
	}

	var generatorClientFactory ring_client.PoolAddrFunc = func(addr string) (ring_client.PoolClient, error) {
		return generator_client.New(addr, generatorClientConfig)
	}

	ingesterPools := make([]*ring_client.Pool, 0, len(ingesterRings))
	for i, ring := range ingesterRings {
		pool := ring_client.NewPool(fmt.Sprintf("querier_pool_%d", i),
			ingesterClientConfig.PoolConfig,
			ring_client.NewRingServiceDiscovery(ring),
			ingesterClientFactory,
			metricIngesterClients,
			log.Logger)
		ingesterPools = append(ingesterPools, pool)
	}

	q := &Querier{
		cfg:           cfg,
		ingesterRings: ingesterRings,
		ingesterPools: ingesterPools,
		generatorRing: generatorRing,
		generatorPool: ring_client.NewPool("querier_to_generator_pool",
			generatorClientConfig.PoolConfig,
			ring_client.NewRingServiceDiscovery(generatorRing),
			generatorClientFactory,
			metricMetricsGeneratorClients,
			log.Logger),
		engine: traceql.NewEngine(),
		store:  store,
		limits: limits,
	}

	q.Service = services.NewBasicService(q.starting, q.running, q.stopping)
	return q, nil
}

func (q *Querier) CreateAndRegisterWorker(handler http.Handler) error {
	q.cfg.Worker.MaxConcurrentRequests = q.cfg.MaxConcurrentQueries
	worker, err := worker.NewQuerierWorker(
		q.cfg.Worker,
		httpgrpc_server.NewServer(handler),
		log.Logger,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create frontend worker: %w", err)
	}

	subservices := []services.Service{worker, q.generatorPool}
	for _, pool := range q.ingesterPools {
		subservices = append(subservices, pool)
	}
	err = q.RegisterSubservices(subservices...)
	if err != nil {
		return fmt.Errorf("failed to register generator pool sub-service: %w", err)
	}

	return nil
}

func (q *Querier) RegisterSubservices(s ...services.Service) error {
	var err error
	q.subservices, err = services.NewManager(s...)
	q.subservicesWatcher = services.NewFailureWatcher()
	q.subservicesWatcher.WatchManager(q.subservices)
	return err
}

func (q *Querier) starting(ctx context.Context) error {
	if q.subservices != nil {
		err := services.StartManagerAndAwaitHealthy(ctx, q.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices: %w", err)
		}
	}

	return nil
}

func (q *Querier) running(ctx context.Context) error {
	if q.subservices != nil {
		select {
		case <-ctx.Done():
			return nil
		case err := <-q.subservicesWatcher.Chan():
			return fmt.Errorf("subservices failed: %w", err)
		}
	} else {
		<-ctx.Done()
	}
	return nil
}

func (q *Querier) stopping(_ error) error {
	if q.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), q.subservices)
	}
	return nil
}

// FindTraceByID implements tempopb.Querier.
func (q *Querier) FindTraceByID(ctx context.Context, req *tempopb.TraceByIDRequest, timeStart int64, timeEnd int64) (*tempopb.TraceByIDResponse, error) {
	if !validation.ValidTraceID(req.TraceID) {
		return nil, errors.New("invalid trace id")
	}

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.FindTraceByID: %w", err)
	}

	ctx, span := tracer.Start(ctx, "Querier.FindTraceByID")
	defer span.End()

	span.SetAttributes(attribute.String("queryMode", req.QueryMode))

	maxBytes := q.limits.MaxBytesPerTrace(userID)
	combiner := trace.NewCombiner(maxBytes, req.AllowPartialTrace)
	var inspectedBytes uint64

	if req.QueryMode == QueryModeIngesters || req.QueryMode == QueryModeAll {
		var getRSFn replicationSetFn
		if q.cfg.QueryRelevantIngesters {
			traceKey := util.TokenFor(userID, req.TraceID)
			getRSFn = func(r ring.ReadRing) (ring.ReplicationSet, error) {
				return r.Get(traceKey, ring.Read, nil, nil, nil)
			}
		}

		// get responses from all ingesters in parallel
		span.AddEvent("searching ingesters")
		forEach := func(funcCtx context.Context, client tempopb.QuerierClient) (any, error) {
			return client.FindTraceByID(funcCtx, req)
		}
		partialTraces, err := q.forIngesterRings(ctx, userID, getRSFn, forEach)
		if err != nil {
			return nil, fmt.Errorf("error querying ingesters in Querier.FindTraceByID: %w", err)
		}

		var spanCountTotal, traceCountTotal int64
		found := false

		for _, partialTrace := range partialTraces {
			resp := partialTrace.(*tempopb.TraceByIDResponse)
			if resp.Trace == nil {
				continue
			}

			found = true

			spanCount, err := combiner.Consume(resp.Trace)
			if err != nil {
				return nil, fmt.Errorf("error combining ingester results in Querier.FindTraceByID: %w", err)
			}

			spanCountTotal += int64(spanCount)
			traceCountTotal++
			if resp.Metrics != nil {
				inspectedBytes += resp.Metrics.InspectedBytes
			}
		}

		span.AddEvent("done searching ingesters", oteltrace.WithAttributes(
			attribute.Bool("found", found),
			attribute.Int64("combinedSpans", spanCountTotal),
			attribute.Int64("combinedTraces", traceCountTotal),
		))
	}

	if req.QueryMode == QueryModeBlocks || req.QueryMode == QueryModeAll {
		span.AddEvent("searching store", oteltrace.WithAttributes(
			attribute.Int64("timeStart", timeStart),
			attribute.Int64("timeEnd", timeEnd),
		))

		opts := common.DefaultSearchOptionsWithMaxBytes(maxBytes)
		opts.BlockReplicationFactor = backend.DefaultReplicationFactor
		partialTraces, blockErrs, err := q.store.Find(ctx, userID, req.TraceID, req.BlockStart, req.BlockEnd, timeStart, timeEnd, opts)
		if err != nil {
			retErr := fmt.Errorf("error querying store in Querier.FindTraceByID: %w", err)
			span.RecordError(retErr)
			return nil, retErr
		}

		if len(blockErrs) > 0 {
			return nil, multierr.Combine(blockErrs...)
		}

		span.AddEvent("done searching store", oteltrace.WithAttributes(
			attribute.Int("foundPartialTraces", len(partialTraces))))

		for _, partialTrace := range partialTraces {
			if partialTrace == nil {
				continue
			}
			_, err = combiner.Consume(partialTrace.Trace)
			if err != nil {
				return nil, err
			}
			if partialTrace.Metrics != nil {
				inspectedBytes += partialTrace.Metrics.InspectedBytes
			}
		}
	}

	completeTrace, _ := combiner.Result()
	resp := &tempopb.TraceByIDResponse{
		Trace:   completeTrace,
		Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: inspectedBytes},
	}

	if combiner.IsPartialTrace() {
		resp.Status = tempopb.TraceByIDResponse_PARTIAL
		resp.Message = fmt.Sprintf("Trace exceeds maximum size of %d bytes, a partial trace is returned", maxBytes)
	}

	return resp, nil
}

// forIngesterRings runs f, in parallel, for given ingesters
func (q *Querier) forIngesterRings(ctx context.Context, userID string, getReplicationSet replicationSetFn, f forEachFn) ([]any, error) {
	if ctx.Err() != nil {
		_ = level.Debug(log.Logger).Log("forIngesterRings context error", "ctx.Err()", ctx.Err().Error())
		return nil, ctx.Err()
	}

	// if we have no configured ingester rings this will fail silently. let's return an actual error instead
	if len(q.ingesterRings) == 0 {
		return nil, errors.New("forIngesterRings: no ingester rings configured")
	}

	// if a nil replicationSetFn is passed, that means to just use a standard Read ring
	if getReplicationSet == nil {
		getReplicationSet = func(r ring.ReadRing) (ring.ReplicationSet, error) {
			return r.GetReplicationSetForOperation(ring.Read)
		}
	}

	var mtx sync.Mutex
	var wg sync.WaitGroup

	var overallErr error
	var overallResults []any

	for i, ingesterRing := range q.ingesterRings {
		if q.cfg.ShuffleShardingIngestersEnabled {
			ingesterRing = ingesterRing.ShuffleShardWithLookback(
				userID,
				q.limits.IngestionTenantShardSize(userID),
				q.cfg.ShuffleShardingIngestersLookbackPeriod,
				time.Now(),
			)
		}

		replicationSet, err := getReplicationSet(ingesterRing)
		if err != nil {
			return nil, fmt.Errorf("forIngesterRings: error getting replication set for ring (%d): %w", i, err)
		}
		pool := q.ingesterPools[i]

		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := forOneIngesterRing(ctx, replicationSet, f, pool, q.cfg.ExtraQueryDelay)
			mtx.Lock()
			defer mtx.Unlock()

			if err != nil {
				overallErr = multierr.Combine(overallErr, err)
				return
			}

			overallResults = append(overallResults, results...)
		}()
	}

	wg.Wait()

	if overallErr != nil {
		return nil, overallErr
	}

	return overallResults, nil
}

func forOneIngesterRing(ctx context.Context, replicationSet ring.ReplicationSet, f forEachFn, pool *ring_client.Pool, extraQueryDelay time.Duration) ([]any, error) {
	ctx, span := tracer.Start(ctx, "Querier.forOneIngesterRing")
	defer span.End()

	doFunc := func(funcCtx context.Context, ingester *ring.InstanceDesc) (interface{}, error) {
		if funcCtx.Err() != nil {
			_ = level.Warn(log.Logger).Log("funcCtx.Err()", funcCtx.Err().Error())
			return nil, funcCtx.Err()
		}

		client, err := pool.GetClientFor(ingester.Addr)
		if err != nil {
			return nil, fmt.Errorf("failed to get client for %s: %w", ingester.Addr, err)
		}

		result, err := f(funcCtx, client.(tempopb.QuerierClient))
		if err != nil {
			return nil, fmt.Errorf("failed to execute f() for %s: %w", ingester.Addr, err)
		}

		return result, nil
	}

	return replicationSet.Do(ctx, extraQueryDelay, doFunc)
}

// forGivenGenerators runs f, in parallel, for given generators
func (q *Querier) forGivenGenerators(ctx context.Context, replicationSet ring.ReplicationSet, f forEachGeneratorFn) ([]any, error) {
	if ctx.Err() != nil {
		_ = level.Debug(log.Logger).Log("foreGivenGenerators context error", "ctx.Err()", ctx.Err().Error())
		return nil, ctx.Err()
	}

	ctx, span := tracer.Start(ctx, "Querier.forGivenGenerators")
	defer span.End()

	doFunc := func(funcCtx context.Context, generator *ring.InstanceDesc) (interface{}, error) {
		if funcCtx.Err() != nil {
			_ = level.Warn(log.Logger).Log("funcCtx.Err()", funcCtx.Err().Error())
			return nil, funcCtx.Err()
		}

		client, err := q.generatorPool.GetClientFor(generator.Addr)
		if err != nil {
			return nil, fmt.Errorf("failed to get client for %s: %w", generator.Addr, err)
		}

		result, err := f(funcCtx, client.(tempopb.MetricsGeneratorClient))
		if err != nil {
			return nil, fmt.Errorf("failed to execute f() for %s: %w", generator.Addr, err)
		}

		return result, nil
	}

	results, err := replicationSet.Do(ctx, q.cfg.ExtraQueryDelay, doFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from generators: %w", err)
	}

	return results, nil
}

func (q *Querier) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchRecent: %w", err)
	}

	results, err := q.forIngesterRings(ctx, userID, nil, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchRecent(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchRecent: %w", err)
	}

	return q.postProcessIngesterSearchResults(req, results), nil
}

func (q *Querier) SearchTagsBlocks(ctx context.Context, req *tempopb.SearchTagsBlockRequest) (*tempopb.SearchTagsResponse, error) {
	v2Response, err := q.internalTagsSearchBlockV2(ctx, req)
	if err != nil {
		return nil, err
	}

	distinctValues := collector.NewDistinctString(0, 0, 0)

	// flatten v2 response
	for _, s := range v2Response.Scopes {
		for _, t := range s.Tags {
			distinctValues.Collect(t)
			if distinctValues.Exceeded() {
				break // stop early
			}
		}
	}

	return &tempopb.SearchTagsResponse{
		TagNames: distinctValues.Strings(),
		Metrics:  v2Response.Metrics,
	}, nil
}

func (q *Querier) SearchTagValuesBlocks(ctx context.Context, req *tempopb.SearchTagValuesBlockRequest) (*tempopb.SearchTagValuesResponse, error) {
	return q.internalTagValuesSearchBlock(ctx, req)
}

func (q *Querier) SearchTagsBlocksV2(ctx context.Context, req *tempopb.SearchTagsBlockRequest) (*tempopb.SearchTagsV2Response, error) {
	return q.internalTagsSearchBlockV2(ctx, req)
}

func (q *Querier) SearchTagValuesBlocksV2(ctx context.Context, req *tempopb.SearchTagValuesBlockRequest) (*tempopb.SearchTagValuesV2Response, error) {
	return q.internalTagValuesSearchBlockV2(ctx, req)
}

func (q *Querier) SearchTags(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTags: %w", err)
	}

	maxDataSize := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewDistinctString(maxDataSize, req.MaxTagsPerScope, req.StaleValuesThreshold)
	var inspectedBytes uint64

	results, err := q.forIngesterRings(ctx, userID, nil, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTags(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTags: %w", err)
	}

outer:
	for _, result := range results {
		resp := result.(*tempopb.SearchTagsResponse)
		if resp.Metrics != nil {
			inspectedBytes += resp.Metrics.InspectedBytes
		}

		for _, tag := range resp.TagNames {
			distinctValues.Collect(tag)
			if distinctValues.Exceeded() {
				break outer
			}
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tags in instance exceeded limit, reduce cardinality or size of tags", "userID", userID, "maxDataSize", maxDataSize, "size", distinctValues.Size())
	}

	return &tempopb.SearchTagsResponse{
		TagNames: distinctValues.Strings(),
		Metrics:  &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes},
	}, nil
}

func (q *Querier) SearchTagsV2(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsV2Response, error) {
	orgID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTags: %w", err)
	}

	maxBytesPerTag := q.limits.MaxBytesPerTagValuesQuery(orgID)
	distinctValues := collector.NewScopedDistinctString(maxBytesPerTag, req.MaxTagsPerScope, req.StaleValuesThreshold)
	var inspectedBytes uint64

	// Get results from all ingesters
	results, err := q.forIngesterRings(ctx, orgID, nil, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTagsV2(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTags: %w", err)
	}

outer:
	for _, result := range results {
		resp := result.(*tempopb.SearchTagsV2Response)
		if resp.Metrics != nil {
			inspectedBytes += resp.Metrics.InspectedBytes
		}

		for _, res := range resp.Scopes {
			for _, tag := range res.Tags {
				if distinctValues.Collect(res.Name, tag) {
					break outer
				}
			}
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "Search tags exceeded limit, reduce cardinality or size of tags", "orgID", orgID, "stopReason", distinctValues.StopReason())
	}

	collected := distinctValues.Strings()
	resp := &tempopb.SearchTagsV2Response{
		Scopes:  make([]*tempopb.SearchTagsV2Scope, 0, len(collected)),
		Metrics: &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes},
	}
	for scope, vals := range collected {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scope,
			Tags: vals,
		})
	}

	return resp, nil
}

func (q *Querier) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTagValues: %w", err)
	}

	maxDataSize := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewDistinctString(maxDataSize, req.MaxTagValues, req.StaleValueThreshold)
	var inspectedBytes uint64

	// Virtual tags values. Get these first.
	for _, v := range search.GetVirtualTagValues(req.TagName) {
		// virtual tags are small so no need to stop early here
		distinctValues.Collect(v)
	}

	results, err := q.forIngesterRings(ctx, userID, nil, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTagValues(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTagValues: %w", err)
	}

outer:
	for _, result := range results {
		resp := result.(*tempopb.SearchTagValuesResponse)
		if resp.Metrics != nil {
			inspectedBytes += resp.Metrics.InspectedBytes
		}

		for _, res := range resp.TagValues {
			distinctValues.Collect(res)
			if distinctValues.Exceeded() {
				break outer
			}
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "Search of tag values exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "orgID", userID, "stopReason", distinctValues.StopReason())
	}

	return &tempopb.SearchTagValuesResponse{
		TagValues: distinctValues.Strings(),
		Metrics:   &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes},
	}, nil
}

func (q *Querier) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTagValues: %w", err)
	}

	maxDataSize := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewDistinctValue(maxDataSize, req.MaxTagValues, req.StaleValueThreshold, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
	var inspectedBytes uint64

	// Virtual tags values. Get these first.
	virtualVals := search.GetVirtualTagValuesV2(req.TagName)
	for _, v := range virtualVals {
		// no need to stop early here, virtual tags are small
		distinctValues.Collect(v)
	}

	// with v2 search we can confidently bail if GetVirtualTagValuesV2 gives us any hits. this doesn't work
	// in v1 search b/c intrinsic tags like "status" are conflated with attributes named "status"
	if virtualVals != nil {
		// no data was read to collect virtual tags so 0 bytesRead
		return valuesToV2Response(distinctValues, 0), nil
	}

	results, err := q.forIngesterRings(ctx, userID, nil, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTagValuesV2(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTagValues: %w", err)
	}

outer:
	for _, result := range results {
		resp := result.(*tempopb.SearchTagValuesV2Response)
		if resp.Metrics != nil {
			inspectedBytes += resp.Metrics.InspectedBytes
		}

		for _, res := range resp.TagValues {
			distinctValues.Collect(*res)
			if distinctValues.Exceeded() {
				break outer
			}
		}
	}

	if distinctValues.Exceeded() {
		_ = level.Warn(log.Logger).Log("msg", "Search of tag values exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "orgID", userID, "stopReason", distinctValues.StopReason())
	}

	return valuesToV2Response(distinctValues, inspectedBytes), nil
}

func (q *Querier) SpanMetricsSummary(ctx context.Context, req *tempopb.SpanMetricsSummaryRequest) (*tempopb.SpanMetricsSummaryResponse, error) {
	genReq := &tempopb.SpanMetricsRequest{
		Query:   req.Query,
		GroupBy: req.GroupBy,
		Start:   req.Start,
		End:     req.End,
		Limit:   0,
	}

	// Get results from all generators
	replicationSet, err := q.generatorRing.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, fmt.Errorf("error finding generators in Querier.SpanMetricsSummary: %w", err)
	}

	results, err := q.forGivenGenerators(ctx, replicationSet, func(ctx context.Context, client tempopb.MetricsGeneratorClient) (any, error) {
		return client.GetMetrics(ctx, genReq)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying generators in Querier.SpanMetricsSummary: %w", err)
	}

	// Combine the results
	yyy := make(map[traceqlmetrics.MetricKeys]*traceqlmetrics.LatencyHistogram)
	xxx := make(map[traceqlmetrics.MetricKeys]*tempopb.SpanMetricsSummary)

	var h *traceqlmetrics.LatencyHistogram
	var s traceqlmetrics.MetricSeries
	for _, result := range results {
		r := result.(*tempopb.SpanMetricsResponse)

		for _, m := range r.Metrics {
			s = protoToMetricSeries(m.Series)
			k := s.MetricKeys()

			if _, ok := xxx[k]; !ok {
				xxx[k] = &tempopb.SpanMetricsSummary{Series: m.Series}
			}

			xxx[k].ErrorSpanCount += m.Errors

			var b [64]int
			for _, l := range m.GetLatencyHistogram() {
				// Reconstitude the bucket
				b[l.Bucket] += int(l.Count)
				// Add to the total
				xxx[k].SpanCount += l.Count
			}

			// Combine the histogram
			h = traceqlmetrics.New(b)
			if _, ok := yyy[k]; !ok {
				yyy[k] = h
			} else {
				yyy[k].Combine(*h)
			}
		}
	}

	for s, h := range yyy {
		xxx[s].P50 = h.Percentile(0.5)
		xxx[s].P90 = h.Percentile(0.9)
		xxx[s].P95 = h.Percentile(0.95)
		xxx[s].P99 = h.Percentile(0.99)
	}

	resp := &tempopb.SpanMetricsSummaryResponse{}
	for _, x := range xxx {
		resp.Summaries = append(resp.Summaries, x)
	}

	return resp, nil
}

func valuesToV2Response(distinctValues *collector.DistinctValue[tempopb.TagValue], bytesRead uint64) *tempopb.SearchTagValuesV2Response {
	resp := &tempopb.SearchTagValuesV2Response{
		Metrics: &tempopb.MetadataMetrics{InspectedBytes: bytesRead},
	}
	for _, v := range distinctValues.Values() {
		v2 := v
		resp.TagValues = append(resp.TagValues, &v2)
	}
	return resp
}

// SearchBlock searches the specified subset of the block for the passed tags.
func (q *Querier) SearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
	if err != nil {
		return nil, err
	}

	enc, err := backend.ParseEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}

	dc, err := backend.DedicatedColumnsFromTempopb(req.DedicatedColumns)
	if err != nil {
		return nil, err
	}

	meta := &backend.BlockMeta{
		Version:          req.Version,
		TenantID:         tenantID,
		Encoding:         enc,
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
		DataEncoding:     req.DataEncoding,
		FooterSize:       req.FooterSize,
		DedicatedColumns: dc,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)
	opts.MaxBytes = q.limits.MaxBytesPerTrace(tenantID)

	if api.IsTraceQLQuery(req.SearchReq) {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return q.store.Fetch(ctx, meta, req, opts)
		})

		return q.engine.ExecuteSearch(ctx, req.SearchReq, fetcher)
	}

	return q.store.Search(ctx, meta, req.SearchReq, opts)
}

func (q *Querier) internalTagsSearchBlockV2(ctx context.Context, req *tempopb.SearchTagsBlockRequest) (*tempopb.SearchTagsV2Response, error) {
	// For the intrinsic scope there is nothing to do in the querier,
	// these are always added by the frontend.
	if req.SearchReq.Scope == api.ParamScopeIntrinsic {
		return &tempopb.SearchTagsV2Response{}, nil
	}

	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
	if err != nil {
		return nil, err
	}

	enc, err := backend.ParseEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}

	dc, err := backend.DedicatedColumnsFromTempopb(req.DedicatedColumns)
	if err != nil {
		return nil, err
	}

	meta := &backend.BlockMeta{
		Version:          req.Version,
		TenantID:         tenantID,
		Encoding:         enc,
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
		DataEncoding:     req.DataEncoding,
		FooterSize:       req.FooterSize,
		DedicatedColumns: dc,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)

	query := traceql.ExtractMatchers(req.SearchReq.Query)
	if traceql.IsEmptyQuery(query) {
		return q.store.SearchTags(ctx, meta, req, opts)
	}

	valueCollector := collector.NewScopedDistinctString(q.limits.MaxBytesPerTagValuesQuery(tenantID), req.MaxTagsPerScope, req.StaleValueThreshold)
	var inspectedBytes uint64

	fetcher := traceql.NewTagNamesFetcherWrapper(func(ctx context.Context, req traceql.FetchTagsRequest, cb traceql.FetchTagsCallback) error {
		return q.store.FetchTagNames(ctx, meta, req, cb, func(bytesRead uint64) { inspectedBytes += bytesRead }, common.DefaultSearchOptions())
	})

	scope := traceql.AttributeScopeFromString(req.SearchReq.Scope)
	if scope == traceql.AttributeScopeUnknown {
		return nil, fmt.Errorf("unknown scope: %s", req.SearchReq.Scope)
	}

	err = q.engine.ExecuteTagNames(ctx, scope, query, func(tag string, scope traceql.AttributeScope) bool {
		return valueCollector.Collect(scope.String(), tag)
	}, fetcher)
	if err != nil {
		return nil, err
	}

	if valueCollector.Exceeded() {
		level.Warn(log.Logger).Log("msg", "Search tags exceeded limit, reduce cardinality or size of tags", "orgID", tenantID, "stopReason", valueCollector.StopReason())
	}

	scopedVals := valueCollector.Strings()
	resp := &tempopb.SearchTagsV2Response{
		Scopes:  make([]*tempopb.SearchTagsV2Scope, 0, len(scopedVals)),
		Metrics: &tempopb.MetadataMetrics{InspectedBytes: inspectedBytes},
	}
	for scope, vals := range scopedVals {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scope,
			Tags: vals,
		})
	}

	return resp, nil
}

func (q *Querier) internalTagValuesSearchBlock(ctx context.Context, req *tempopb.SearchTagValuesBlockRequest) (*tempopb.SearchTagValuesResponse, error) {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return &tempopb.SearchTagValuesResponse{}, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
	if err != nil {
		return &tempopb.SearchTagValuesResponse{}, err
	}

	enc, err := backend.ParseEncoding(req.Encoding)
	if err != nil {
		return &tempopb.SearchTagValuesResponse{}, err
	}

	dc, err := backend.DedicatedColumnsFromTempopb(req.DedicatedColumns)
	if err != nil {
		return &tempopb.SearchTagValuesResponse{}, err
	}

	meta := &backend.BlockMeta{
		Version:          req.Version,
		TenantID:         tenantID,
		Encoding:         enc,
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
		DataEncoding:     req.DataEncoding,
		FooterSize:       req.FooterSize,
		DedicatedColumns: dc,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)

	resp, err := q.store.SearchTagValues(ctx, meta, req, opts)
	if err != nil {
		return &tempopb.SearchTagValuesResponse{}, err
	}

	return resp, nil
}

func (q *Querier) internalTagValuesSearchBlockV2(ctx context.Context, req *tempopb.SearchTagValuesBlockRequest) (*tempopb.SearchTagValuesV2Response, error) {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return &tempopb.SearchTagValuesV2Response{}, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
	if err != nil {
		return &tempopb.SearchTagValuesV2Response{}, err
	}

	enc, err := backend.ParseEncoding(req.Encoding)
	if err != nil {
		return &tempopb.SearchTagValuesV2Response{}, err
	}

	dc, err := backend.DedicatedColumnsFromTempopb(req.DedicatedColumns)
	if err != nil {
		return &tempopb.SearchTagValuesV2Response{}, err
	}

	meta := &backend.BlockMeta{
		Version:          req.Version,
		TenantID:         tenantID,
		Encoding:         enc,
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
		DataEncoding:     req.DataEncoding,
		FooterSize:       req.FooterSize,
		DedicatedColumns: dc,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)

	query := traceql.ExtractMatchers(req.SearchReq.Query)
	if traceql.IsEmptyQuery(query) {
		return q.store.SearchTagValuesV2(ctx, meta, req.SearchReq, opts)
	}

	tag, err := traceql.ParseIdentifier(req.SearchReq.TagName)
	if err != nil {
		return nil, err
	}

	valueCollector := collector.NewDistinctValue(q.limits.MaxBytesPerTagValuesQuery(tenantID),
		req.SearchReq.MaxTagValues, req.SearchReq.StaleValueThreshold,
		func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

	var inspectedBytes uint64

	fetcher := traceql.NewTagValuesFetcherWrapper(func(ctx context.Context, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback) error {
		return q.store.FetchTagValues(ctx, meta, req, cb, func(bytesRead uint64) { inspectedBytes += bytesRead }, opts)
	})

	err = q.engine.ExecuteTagValues(ctx, tag, query, traceql.MakeCollectTagValueFunc(valueCollector.Collect), fetcher)
	if err != nil {
		return nil, err
	}

	if valueCollector.Exceeded() {
		level.Warn(log.Logger).Log("msg", "Search tags exceeded limit, reduce cardinality or size of tags", "orgID", tenantID, "stopReason", valueCollector.StopReason())
	}

	return valuesToV2Response(valueCollector, inspectedBytes), nil
}

func (q *Querier) postProcessIngesterSearchResults(req *tempopb.SearchRequest, results []any) *tempopb.SearchResponse {
	response := &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}

	traces := map[string]*tempopb.TraceSearchMetadata{}

	for _, result := range results {
		sr := result.(*tempopb.SearchResponse)

		for _, t := range sr.Traces {
			// Just simply take first result for each trace
			if _, ok := traces[t.TraceID]; !ok {
				traces[t.TraceID] = t
			}
		}
		if sr.Metrics != nil {
			response.Metrics.InspectedBytes += sr.Metrics.InspectedBytes
			response.Metrics.InspectedTraces += sr.Metrics.InspectedTraces
		}
	}

	for _, t := range traces {
		response.Traces = append(response.Traces, t)
	}

	// Sort and limit results
	sort.Slice(response.Traces, func(i, j int) bool {
		return response.Traces[i].StartTimeUnixNano > response.Traces[j].StartTimeUnixNano
	})
	if req.Limit != 0 && int(req.Limit) < len(response.Traces) {
		response.Traces = response.Traces[:req.Limit]
	}

	return response
}

func protoToMetricSeries(proto []*tempopb.KeyValue) traceqlmetrics.MetricSeries {
	r := traceqlmetrics.MetricSeries{}
	for i := range proto {
		r[i] = protoToTraceQLStatic(proto[i])
	}
	return r
}

func protoToTraceQLStatic(kv *tempopb.KeyValue) traceqlmetrics.KeyValue {
	var val traceql.Static

	switch traceql.StaticType(kv.Value.Type) {
	case traceql.TypeInt:
		val = traceql.NewStaticInt(int(kv.Value.N))
	case traceql.TypeFloat:
		val = traceql.NewStaticFloat(kv.Value.F)
	case traceql.TypeString:
		val = traceql.NewStaticString(kv.Value.S)
	case traceql.TypeBoolean:
		val = traceql.NewStaticBool(kv.Value.B)
	case traceql.TypeDuration:
		val = traceql.NewStaticDuration(time.Duration(kv.Value.D))
	case traceql.TypeStatus:
		val = traceql.NewStaticStatus(traceql.Status(kv.Value.Status))
	case traceql.TypeKind:
		val = traceql.NewStaticKind(traceql.Kind(kv.Value.Kind))
	default:
		val = traceql.NewStaticNil()
	}

	return traceqlmetrics.KeyValue{
		Key:   kv.Key,
		Value: val,
	}
}
