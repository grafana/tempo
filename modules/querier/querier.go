package querier

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	httpgrpc_server "github.com/weaveworks/common/httpgrpc/server"
	"github.com/weaveworks/common/user"
	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"

	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier/worker"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/hedgedmetrics"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var (
	metricIngesterClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "querier_ingester_clients",
		Help:      "The current number of ingester clients.",
	})
	metricEndpointDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "querier_external_endpoint_duration_seconds",
		Help:      "The duration of the external endpoints.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"endpoint"})
	metricExternalHedgedRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "querier_external_endpoint_hedged_roundtrips_total",
			Help:      "Total number of hedged external requests. Registered as a gauge for code sanity. This is a counter.",
		},
	)
)

// Querier handlers queries.
type Querier struct {
	services.Service

	cfg    Config
	ring   ring.ReadRing
	pool   *ring_client.Pool
	engine *traceql.Engine
	store  storage.Store
	limits *overrides.Overrides

	searchClient     *http.Client
	searchPreferSelf *semaphore.Weighted

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

type responseFromIngesters struct {
	addr     string
	response interface{}
}

// New makes a new Querier.
func New(cfg Config, clientCfg ingester_client.Config, ring ring.ReadRing, store storage.Store, limits *overrides.Overrides) (*Querier, error) {
	// TODO should we somehow refuse traceQL queries if backend encoding is not parquet?

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
			log.Logger),
		engine:           traceql.NewEngine(),
		store:            store,
		limits:           limits,
		searchPreferSelf: semaphore.NewWeighted(int64(cfg.Search.PreferSelf)),
		searchClient:     http.DefaultClient,
	}

	//
	if cfg.Search.HedgeRequestsAt != 0 {
		var err error
		var stats *hedgedhttp.Stats
		q.searchClient, stats, err = hedgedhttp.NewClientAndStats(cfg.Search.HedgeRequestsAt, cfg.Search.HedgeRequestsUpTo, http.DefaultClient)
		if err != nil {
			return nil, err
		}
		hedgedmetrics.Publish(stats, metricExternalHedgedRequests)
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

	return q.RegisterSubservices(worker, q.pool)
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
			return fmt.Errorf("failed to start subservices %w", err)
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
			return fmt.Errorf("subservices failed %w", err)
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
		return nil, fmt.Errorf("invalid trace id")
	}

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.FindTraceByID")
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.FindTraceByID")
	defer span.Finish()

	span.SetTag("queryMode", req.QueryMode)

	combiner := trace.NewCombiner()
	var spanCount, spanCountTotal, traceCountTotal int
	if req.QueryMode == QueryModeIngesters || req.QueryMode == QueryModeAll {
		var replicationSet ring.ReplicationSet
		var err error
		if q.cfg.QueryRelevantIngesters {
			traceKey := util.TokenFor(userID, req.TraceID)
			replicationSet, err = q.ring.Get(traceKey, ring.Read, nil, nil, nil)
		} else {
			replicationSet, err = q.ring.GetReplicationSetForOperation(ring.Read)
		}
		if err != nil {
			return nil, errors.Wrap(err, "error finding ingesters in Querier.FindTraceByID")
		}

		span.LogFields(ot_log.String("msg", "searching ingesters"))

		forEachFunc := func(funcCtx context.Context, client tempopb.QuerierClient) (interface{}, error) {
			return client.FindTraceByID(funcCtx, req)
		}

		// get responses from all ingesters in parallel
		responses, err := q.forGivenIngesters(ctx, replicationSet, forEachFunc)
		if err != nil {
			return nil, errors.Wrap(err, "error querying ingesters in Querier.FindTraceByID")
		}

		found := false
		for _, r := range responses {
			t := r.response.(*tempopb.TraceByIDResponse).Trace
			if t != nil {
				spanCount = combiner.Consume(t)
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

	var failedBlocks int
	if req.QueryMode == QueryModeBlocks || req.QueryMode == QueryModeAll {
		span.LogFields(ot_log.String("msg", "searching store"))
		span.LogFields(ot_log.String("timeStart", fmt.Sprint(timeStart)))
		span.LogFields(ot_log.String("timeEnd", fmt.Sprint(timeEnd)))
		partialTraces, blockErrs, err := q.store.Find(ctx, userID, req.TraceID, req.BlockStart, req.BlockEnd, timeStart, timeEnd)
		if err != nil {
			retErr := errors.Wrap(err, "error querying store in Querier.FindTraceByID")
			ot_log.Error(retErr)
			return nil, retErr
		}

		if blockErrs != nil {
			failedBlocks = len(blockErrs)
			span.LogFields(
				ot_log.Int("failedBlocks", failedBlocks),
				ot_log.Error(multierr.Combine(blockErrs...)),
			)
			_ = level.Warn(log.Logger).Log("msg", fmt.Sprintf("failed to query %d blocks", failedBlocks), "blockErrs", multierr.Combine(blockErrs...))
		}

		span.LogFields(
			ot_log.String("msg", "done searching store"),
			ot_log.Int("foundPartialTraces", len(partialTraces)))

		for _, partialTrace := range partialTraces {
			combiner.Consume(partialTrace)
		}
	}

	completeTrace, _ := combiner.Result()

	return &tempopb.TraceByIDResponse{
		Trace: completeTrace,
		Metrics: &tempopb.TraceByIDMetrics{
			FailedBlocks: uint32(failedBlocks),
		},
	}, nil
}

// forGivenIngesters runs f, in parallel, for given ingesters
func (q *Querier) forGivenIngesters(ctx context.Context, replicationSet ring.ReplicationSet, f func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error)) ([]responseFromIngesters, error) {
	if ctx.Err() != nil {
		_ = level.Debug(log.Logger).Log("foreGivenIngesters context error", "ctx.Err()", ctx.Err().Error())
		return nil, ctx.Err()
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.forGivenIngesters")
	defer span.Finish()

	doFunc := func(funcCtx context.Context, ingester *ring.InstanceDesc) (interface{}, error) {
		if funcCtx.Err() != nil {
			_ = level.Warn(log.Logger).Log("funcCtx.Err()", funcCtx.Err().Error())
			return nil, funcCtx.Err()
		}

		client, err := q.pool.GetClientFor(ingester.Addr)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get client for %s", ingester.Addr))
		}

		resp, err := f(funcCtx, client.(tempopb.QuerierClient))
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to execute f() for %s", ingester.Addr))
		}

		return responseFromIngesters{ingester.Addr, resp}, nil
	}

	results, err := replicationSet.Do(ctx, q.cfg.ExtraQueryDelay, doFunc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response from ingesters")
	}

	responses := make([]responseFromIngesters, 0, len(results))
	for _, result := range results {
		responses = append(responses, result.(responseFromIngesters))
	}

	return responses, nil
}

func (q *Querier) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	_, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.Search")
	}

	replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.Search")
	}

	responses, err := q.forGivenIngesters(ctx, replicationSet, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchRecent(ctx, req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.Search")
	}

	return q.postProcessIngesterSearchResults(req, responses), nil
}

func (q *Querier) SearchTags(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.SearchTags")
	}

	limit := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	// Get results from all ingesters
	replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.SearchTags")
	}
	lookupResults, err := q.forGivenIngesters(ctx, replicationSet, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTags(ctx, req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.SearchTags")
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

func (q *Querier) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.SearchTagValues")
	}

	limit := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	// Virtual tags values. Get these first.
	for _, v := range search.GetVirtualTagValues(req.TagName) {
		distinctValues.Collect(v)
	}

	// Get results from all ingesters
	replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.SearchTagValues")
	}
	lookupResults, err := q.forGivenIngesters(ctx, replicationSet, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTagValues(ctx, req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.SearchTagValues")
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
		return nil, errors.Wrap(err, "error extracting org id in Querier.SearchTagValues")
	}

	limit := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctValueCollector(limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

	// Virtual tags values. Get these first.
	for _, v := range search.GetVirtualTagValuesV2(req.TagName) {
		distinctValues.Collect(v)
	}

	// Get results from all ingesters
	replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.SearchTagValues")
	}
	lookupResults, err := q.forGivenIngesters(ctx, replicationSet, func(ctx context.Context, client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTagValuesV2(ctx, req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.SearchTagValues")
	}
	for _, resp := range lookupResults {
		for _, res := range resp.response.(*tempopb.SearchTagValuesV2Response).TagValues {
			distinctValues.Collect(*res)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	resp := &tempopb.SearchTagValuesV2Response{}

	for _, v := range distinctValues.Values() {
		v2 := v
		resp.TagValues = append(resp.TagValues, &v2)
	}

	return resp, nil
}

// SearchBlock searches the specified subset of the block for the passed tags.
func (q *Querier) SearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	// if we have no external configuration always search in the querier
	if len(q.cfg.Search.ExternalEndpoints) == 0 {
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
		return nil, errors.Wrap(err, "error extracting org id for externalEndpoint")
	}
	maxBytes := q.limits.MaxBytesPerTrace(tenantID)

	endpoint := q.cfg.Search.ExternalEndpoints[rand.Intn(len(q.cfg.Search.ExternalEndpoints))]
	return q.searchExternalEndpoint(ctx, endpoint, maxBytes, req)
}

func (q *Querier) internalSearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.BackendSearch")
	}

	blockID, err := uuid.Parse(req.BlockID)
	if err != nil {
		return nil, err
	}

	enc, err := backend.ParseEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}

	meta := &backend.BlockMeta{
		Version:       req.Version,
		TenantID:      tenantID,
		Encoding:      enc,
		Size:          req.Size_,
		IndexPageSize: req.IndexPageSize,
		TotalRecords:  req.TotalRecords,
		BlockID:       blockID,
		DataEncoding:  req.DataEncoding,
		FooterSize:    req.FooterSize,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)
	opts.MaxBytes = q.limits.MaxBytesPerTrace(tenantID)

	if api.IsTraceQLQuery(req.SearchReq) {
		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return q.store.Fetch(ctx, meta, req, opts)
		})

		return q.engine.Execute(ctx, req.SearchReq, fetcher)
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
			response.Metrics.InspectedBlocks += sr.Metrics.InspectedBlocks
			response.Metrics.SkippedBlocks += sr.Metrics.SkippedBlocks
		}
	}

	for _, t := range traces {
		if t.RootServiceName == "" {
			t.RootServiceName = search.RootSpanNotYetReceivedText
		}
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

func (q *Querier) searchExternalEndpoint(ctx context.Context, externalEndpoint string, maxBytes int, searchReq *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	req, err := http.NewRequest(http.MethodGet, externalEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to make new request: %w", err)
	}
	req, err = api.BuildSearchBlockRequest(req, searchReq)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to build search block request: %w", err)
	}
	req = api.AddServerlessParams(req, maxBytes)
	err = user.InjectOrgIDIntoHTTPRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to inject tenant id: %w", err)
	}
	start := time.Now()
	resp, err := q.searchClient.Do(req)
	metricEndpointDuration.WithLabelValues(externalEndpoint).Observe(time.Since(start).Seconds())
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to call http: %s, %w", externalEndpoint, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external endpoint returned %d, %s", resp.StatusCode, string(body))
	}
	var searchResp tempopb.SearchResponse
	err = jsonpb.Unmarshal(bytes.NewReader(body), &searchResp)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to unmarshal body: %s, %w", string(body), err)
	}
	return &searchResp, nil
}
