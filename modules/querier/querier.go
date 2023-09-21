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
	"github.com/google/uuid"
	httpgrpc_server "github.com/grafana/dskit/httpgrpc/server"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"

	generator_client "github.com/grafana/tempo/modules/generator/client"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier/external"
	"github.com/grafana/tempo/modules/querier/worker"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
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

// jpe - ditch unretyrable error
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

	externalClient *external.Client

	searchPreferSelf *semaphore.Weighted

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

type responseFromIngesters struct {
	addr     string
	response interface{}
}

type responseFromGenerators struct {
	addr     string
	response interface{}
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
	ingesterClientFactory := func(addr string) (ring_client.PoolClient, error) {
		return ingester_client.New(addr, ingesterClientConfig)
	}

	generatorClientFactory := func(addr string) (ring_client.PoolClient, error) {
		return generator_client.New(addr, generatorClientConfig)
	}

	externalClient, err := external.NewClient(&external.Config{
		Backend:        cfg.Search.ExternalBackend,
		CloudRunConfig: cfg.Search.CloudRun,

		HedgeRequestsAt:   cfg.Search.HedgeRequestsAt,
		HedgeRequestsUpTo: cfg.Search.HedgeRequestsUpTo,

		HTTPConfig: &external.HTTPConfig{
			Endpoints: cfg.Search.ExternalEndpoints,
		},
	})
	if err != nil {
		return nil, err
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
		engine:           traceql.NewEngine(),
		store:            store,
		limits:           limits,
		searchPreferSelf: semaphore.NewWeighted(int64(cfg.Search.PreferSelf)),
		externalClient:   externalClient,
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

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.FindTraceByID")
	defer span.Finish()

	span.SetTag("queryMode", req.QueryMode)

	maxBytes := q.limits.MaxBytesPerTrace(userID)
	combiner := trace.NewCombiner(maxBytes)

	var spanCount, spanCountTotal, traceCountTotal int
	if req.QueryMode == QueryModeIngesters || req.QueryMode == QueryModeAll {
		var getRSFn replicationSetFn
		if q.cfg.QueryRelevantIngesters {
			traceKey := util.TokenFor(userID, req.TraceID)
			getRSFn = func(r ring.ReadRing) (ring.ReplicationSet, error) {
				return r.Get(traceKey, ring.Read, nil, nil, nil)
			}
		}

		// get responses from all ingesters in parallel
		span.LogFields(ot_log.String("msg", "searching ingesters"))
		responses, err := q.forIngesterRings(ctx, getRSFn, func(funcCtx context.Context, client tempopb.QuerierClient) (interface{}, error) {
			return client.FindTraceByID(funcCtx, req)
		})
		if err != nil {
			return nil, fmt.Errorf("error querying ingesters in Querier.FindTraceByID: %w", err)
		}

		found := false
		for _, r := range responses {
			t := r.response.(*tempopb.TraceByIDResponse).Trace
			if t != nil {
				spanCount, err = combiner.Consume(t)
				if err != nil {
					return nil, err
				}

				spanCountTotal += spanCount
				traceCountTotal++
				found = true
			}
		}
		span.LogFields(ot_log.String("msg", "done searching ingesters"),
			ot_log.Bool("found", found),
			ot_log.Int("combinedSpans", spanCountTotal),
			ot_log.Int("combinedTraces", traceCountTotal))
	}

	if req.QueryMode == QueryModeBlocks || req.QueryMode == QueryModeAll {
		span.LogFields(ot_log.String("msg", "searching store"))
		span.LogFields(ot_log.String("timeStart", fmt.Sprint(timeStart)))
		span.LogFields(ot_log.String("timeEnd", fmt.Sprint(timeEnd)))

		opts := common.DefaultSearchOptionsWithMaxBytes(maxBytes)
		partialTraces, blockErrs, err := q.store.Find(ctx, userID, req.TraceID, req.BlockStart, req.BlockEnd, timeStart, timeEnd, opts)
		if err != nil {
			retErr := fmt.Errorf("error querying store in Querier.FindTraceByID: %w", err)
			ot_log.Error(retErr)
			return nil, retErr
		}

		if len(blockErrs) > 0 {
			return nil, multierr.Combine(blockErrs...)
		}

		span.LogFields(
			ot_log.String("msg", "done searching store"),
			ot_log.Int("foundPartialTraces", len(partialTraces)))

		for _, partialTrace := range partialTraces {
			_, err = combiner.Consume(partialTrace)
			if err != nil {
				return nil, err
			}
		}
	}

	completeTrace, _ := combiner.Result()

	return &tempopb.TraceByIDResponse{
		Trace:   completeTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
	}, nil
}

type (
	forEachFn        func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error)
	replicationSetFn func(r ring.ReadRing) (ring.ReplicationSet, error)
)

// forIngesterRings runs f, in parallel, for given ingesters
func (q *Querier) forIngesterRings(ctx context.Context, getReplicationSet replicationSetFn, f forEachFn) ([]responseFromIngesters, error) {
	if ctx.Err() != nil {
		_ = level.Debug(log.Logger).Log("forIngesterRings context error", "ctx.Err()", ctx.Err().Error())
		return nil, ctx.Err()
	}

	// if we have no configured ingester rings this will fail silently. let's return an actual error instead
	if len(q.ingesterRings) == 0 {
		return nil, errors.New("forIngesterRings: no ingester rings configured")
	}

	// if a nil replicationsetfn is passed, that means to just use a standard readring
	if getReplicationSet == nil {
		getReplicationSet = func(r ring.ReadRing) (ring.ReplicationSet, error) {
			return r.GetReplicationSetForOperation(ring.Read)
		}
	}

	var mtx sync.Mutex
	var wg sync.WaitGroup

	var responses []responseFromIngesters
	var responseErr error

	for i, ring := range q.ingesterRings {
		replicationSet, err := getReplicationSet(ring)
		if err != nil {
			return nil, fmt.Errorf("forIngesterRings: error getting replication set for ring (%d): %w", i, err)
		}
		pool := q.ingesterPools[i]

		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := forOneIngesterRing(ctx, replicationSet, f, pool, q.cfg.ExtraQueryDelay)

			mtx.Lock()
			defer mtx.Unlock()

			if err != nil {
				responseErr = multierr.Combine(responseErr, err)
				return
			}

			for _, r := range res {
				responses = append(responses, r.(responseFromIngesters))
			}
		}()
	}

	wg.Wait()

	if responseErr != nil {
		return nil, responseErr
	}

	return responses, nil
}

func forOneIngesterRing(ctx context.Context, replicationSet ring.ReplicationSet, f forEachFn, pool *ring_client.Pool, extraQueryDelay time.Duration) ([]interface{}, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.forOneIngester")
	defer span.Finish()

	doFunc := func(funcCtx context.Context, ingester *ring.InstanceDesc) (interface{}, error) {
		if funcCtx.Err() != nil {
			_ = level.Warn(log.Logger).Log("funcCtx.Err()", funcCtx.Err().Error())
			return nil, funcCtx.Err()
		}

		client, err := pool.GetClientFor(ingester.Addr)
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("failed to get client for %s", ingester.Addr), "%w", err)
		}

		resp, err := f(funcCtx, client.(tempopb.QuerierClient))
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("failed to execute f() for %s", ingester.Addr), "%w", err)
		}

		return responseFromIngesters{ingester.Addr, resp}, nil
	}

	return replicationSet.Do(ctx, extraQueryDelay, doFunc)
}

// forGivenGenerators runs f, in parallel, for given generators
func (q *Querier) forGivenGenerators(
	ctx context.Context,
	replicationSet ring.ReplicationSet,
	f func(ctx context.Context, client tempopb.MetricsGeneratorClient) (interface{}, error),
) ([]responseFromGenerators, error) {
	if ctx.Err() != nil {
		_ = level.Debug(log.Logger).Log("foreGivenGenerators context error", "ctx.Err()", ctx.Err().Error())
		return nil, ctx.Err()
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.forGivenGenerators")
	defer span.Finish()

	doFunc := func(funcCtx context.Context, generator *ring.InstanceDesc) (interface{}, error) {
		if funcCtx.Err() != nil {
			_ = level.Warn(log.Logger).Log("funcCtx.Err()", funcCtx.Err().Error())
			return nil, funcCtx.Err()
		}

		client, err := q.generatorPool.GetClientFor(generator.Addr)
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("failed to get client for %s", generator.Addr), "%w", err)
		}

		resp, err := f(funcCtx, client.(tempopb.MetricsGeneratorClient))
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("failed to execute f() for %s", generator.Addr), "%w", err)
		}

		return responseFromGenerators{generator.Addr, resp}, nil
	}

	results, err := replicationSet.Do(ctx, q.cfg.ExtraQueryDelay, doFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from generators: %w", err)
	}

	responses := make([]responseFromGenerators, 0, len(results))
	for _, result := range results {
		responses = append(responses, result.(responseFromGenerators))
	}

	return responses, nil
}

func (q *Querier) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	_, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.Search: %w", err)
	}

	responses, err := q.forIngesterRings(ctx, nil, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchRecent(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.Search: %w", err)
	}

	return q.postProcessIngesterSearchResults(req, responses), nil
}

func (q *Querier) SearchTags(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTags: %w", err)
	}

	limit := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	lookupResults, err := q.forIngesterRings(ctx, nil, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTags(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTags: %w", err)
	}
	for _, resp := range lookupResults {
		for _, res := range resp.response.(*tempopb.SearchTagsResponse).TagNames {
			distinctValues.Collect(res)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tags in instance exceeded limit, reduce cardinality or size of tags", "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	resp := &tempopb.SearchTagsResponse{
		TagNames: distinctValues.Strings(),
	}

	return resp, nil
}

func (q *Querier) SearchTagsV2(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsV2Response, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTags: %w", err)
	}

	// Get results from all ingesters
	lookupResults, err := q.forIngesterRings(ctx, nil, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTagsV2(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTags: %w", err)
	}

	limit := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := map[string]*util.DistinctStringCollector{}

	for _, resp := range lookupResults {
		for _, res := range resp.response.(*tempopb.SearchTagsV2Response).Scopes {
			dvc := distinctValues[res.Name]
			if dvc == nil {
				dvc = util.NewDistinctStringCollector(limit)
				distinctValues[res.Name] = dvc
			}

			for _, tag := range res.Tags {
				dvc.Collect(tag)
			}
		}
	}

	for scope, dvc := range distinctValues {
		if dvc.Exceeded() {
			level.Warn(log.Logger).Log("msg", "size of tags in instance exceeded limit, reduce cardinality or size of tags", "userID", userID, "limit", limit, "scope", scope, "total", dvc.TotalDataSize())
		}
	}

	resp := &tempopb.SearchTagsV2Response{}
	for scope, dvc := range distinctValues {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scope,
			Tags: dvc.Strings(),
		})
	}

	return resp, nil
}

func (q *Querier) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTagValues: %w", err)
	}

	limit := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	// Virtual tags values. Get these first.
	for _, v := range search.GetVirtualTagValues(req.TagName) {
		distinctValues.Collect(v)
	}

	lookupResults, err := q.forIngesterRings(ctx, nil, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTagValues(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTagValues: %w", err)
	}
	for _, resp := range lookupResults {
		for _, res := range resp.response.(*tempopb.SearchTagValuesResponse).TagValues {
			distinctValues.Collect(res)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	resp := &tempopb.SearchTagValuesResponse{
		TagValues: distinctValues.Strings(),
	}

	return resp, nil
}

func (q *Querier) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTagValues: %w", err)
	}

	limit := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctValueCollector(limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

	// Virtual tags values. Get these first.
	virtualVals := search.GetVirtualTagValuesV2(req.TagName)
	for _, v := range virtualVals {
		distinctValues.Collect(v)
	}

	// with v2 search we can confidently bail if GetVirtualTagValuesV2 gives us any hits. this doesn't work
	// in v1 search b/c intrinsic tags like "status" are conflated with attributes named "status"
	if virtualVals != nil {
		return valuesToV2Response(distinctValues), nil
	}

	// Get results from all ingesters
	lookupResults, err := q.forIngesterRings(ctx, nil, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTagValuesV2(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying ingesters in Querier.SearchTagValues: %w", err)
	}
	for _, resp := range lookupResults {
		for _, res := range resp.response.(*tempopb.SearchTagValuesV2Response).TagValues {
			distinctValues.Collect(*res)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	return valuesToV2Response(distinctValues), nil
}

func (q *Querier) SpanMetricsSummary(
	ctx context.Context,
	req *tempopb.SpanMetricsSummaryRequest,
) (*tempopb.SpanMetricsSummaryResponse, error) {
	// userID, err := user.ExtractOrgID(ctx)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "error extracting org id in Querier.SpanMetricsSummary")
	// }

	// limit := q.limits.MaxBytesPerTagValuesQuery(userID)

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
	lookupResults, err := q.forGivenGenerators(
		ctx,
		replicationSet,
		func(ctx context.Context, client tempopb.MetricsGeneratorClient) (interface{}, error) {
			return client.GetMetrics(ctx, genReq)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error querying generators in Querier.SpanMetricsSummary: %w", err)
	}

	// Assemble the results from the generators in the pool
	results := make([]*tempopb.SpanMetricsResponse, 0, len(lookupResults))
	for _, result := range lookupResults {
		results = append(results, result.response.(*tempopb.SpanMetricsResponse))
	}

	// Combine the results
	yyy := make(map[traceqlmetrics.MetricSeries]*traceqlmetrics.LatencyHistogram)
	xxx := make(map[traceqlmetrics.MetricSeries]*tempopb.SpanMetricsSummary)

	var h *traceqlmetrics.LatencyHistogram
	var s traceqlmetrics.MetricSeries
	for _, r := range results {
		for _, m := range r.Metrics {
			s = protoToMetricSeries(m.Series)

			if _, ok := xxx[s]; !ok {
				xxx[s] = &tempopb.SpanMetricsSummary{Series: m.Series}
			}

			xxx[s].ErrorSpanCount += m.Errors

			var b [64]int
			for _, l := range m.GetLatencyHistogram() {
				// Reconstitude the bucket
				b[l.Bucket] += int(l.Count)
				// Add to the total
				xxx[s].SpanCount += l.Count
			}

			// Combine the histogram
			h = traceqlmetrics.New(b)
			if _, ok := yyy[s]; !ok {
				yyy[s] = h
			} else {
				yyy[s].Combine(*h)
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

func valuesToV2Response(distinctValues *util.DistinctValueCollector[tempopb.TagValue]) *tempopb.SearchTagValuesV2Response {
	resp := &tempopb.SearchTagValuesV2Response{}
	for _, v := range distinctValues.Values() {
		v2 := v
		resp.TagValues = append(resp.TagValues, &v2)
	}

	return resp
}

// SearchBlock searches the specified subset of the block for the passed tags.
func (q *Querier) SearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	// if we have no external configuration always search in the querier
	if q.cfg.Search.ExternalBackend == "" && len(q.cfg.Search.ExternalEndpoints) == 0 {
		return q.internalSearchBlock(ctx, req)
	}

	// if we have external configuration but there's an open slot locally then search in the querier
	if q.searchPreferSelf.TryAcquire(1) {
		defer q.searchPreferSelf.Release(1)
		return q.internalSearchBlock(ctx, req)
	}

	// proxy externally!
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id for externalEndpoint: %w", err)
	}
	maxBytes := q.limits.MaxBytesPerTrace(tenantID)

	return q.externalClient.Search(ctx, maxBytes, req)
}

func (q *Querier) internalSearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := uuid.Parse(req.BlockID)
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
		Size:             req.Size_,
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

func (q *Querier) postProcessIngesterSearchResults(req *tempopb.SearchRequest, rr []responseFromIngesters) *tempopb.SearchResponse {
	response := &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}

	traces := map[string]*tempopb.TraceSearchMetadata{}

	for _, r := range rr {
		sr := r.response.(*tempopb.SearchResponse)
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

func protoToTraceQLStatic(proto *tempopb.KeyValue) traceqlmetrics.KeyValue {
	return traceqlmetrics.KeyValue{
		Key: proto.Key,
		Value: traceql.Static{
			Type:   traceql.StaticType(proto.Value.Type),
			N:      int(proto.Value.N),
			F:      proto.Value.F,
			S:      proto.Value.S,
			B:      proto.Value.B,
			D:      time.Duration(proto.Value.D),
			Status: traceql.Status(proto.Value.Status),
			Kind:   traceql.Kind(proto.Value.Kind),
		},
	}
}
