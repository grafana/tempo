package querier

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-kit/log/level"
	httpgrpc_server "github.com/grafana/dskit/httpgrpc/server"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"

	livestore_client "github.com/grafana/tempo/modules/livestore/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier/external"
	"github.com/grafana/tempo/modules/querier/worker"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/traceqlmetrics"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var tracer = otel.Tracer("modules/querier")

var metricMetricsLiveStoreClients = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "tempo",
	Name:      "querier_livestore_clients",
	Help:      "The current number of livestore clients.",
})

type (
	forEachFn        func(ctx context.Context, client tempopb.QuerierClient) (any, error)
	forEachMetricsFn func(ctx context.Context, client tempopb.MetricsClient) (any, error)
)

// Querier handlers queries.
type Querier struct {
	services.Service

	cfg Config

	liveStorePool *ring_client.Pool
	partitionRing *ring.PartitionInstanceRing

	engine *traceql.Engine
	store  storage.Store
	limits overrides.Interface

	externalClient *external.Client

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New makes a new Querier.
//
// Recent data is queried via the partition ring.
func New(
	cfg Config,

	liveStoreRing ring.ReadRing,
	liveStoreClientConfig livestore_client.Config,
	partitionRing *ring.PartitionInstanceRing,

	queryExternal bool,
	store storage.Store,
	limits overrides.Interface,
) (*Querier, error) {
	var liveStoreClientFactory ring_client.PoolAddrFunc = func(addr string) (ring_client.PoolClient, error) {
		return livestore_client.New(addr, liveStoreClientConfig)
	}

	q := &Querier{
		cfg:           cfg,
		partitionRing: partitionRing,
		liveStorePool: ring_client.NewPool("querier_to_livestore_pool",
			liveStoreClientConfig.PoolConfig,
			ring_client.NewRingServiceDiscovery(liveStoreRing),
			liveStoreClientFactory,
			metricMetricsLiveStoreClients,
			log.Logger),
		engine: traceql.NewEngine(),
		store:  store,
		limits: limits,
	}

	if queryExternal {
		externalClient, err := external.NewClient(cfg.TraceByID.External.Endpoint, cfg.TraceByID.External.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create external client: %w", err)
		}
		q.externalClient = externalClient
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

	subservices := []services.Service{worker, q.liveStorePool}
	err = q.RegisterSubservices(subservices...)
	if err != nil {
		return fmt.Errorf("failed to register live-store pool sub-service: %w", err)
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

	userID, err := validation.ExtractValidTenantID(ctx)
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
		// Get responses from all live stores in parallel.
		span.AddEvent("searching live-stores")
		forEach := func(funcCtx context.Context, client tempopb.QuerierClient) (any, error) {
			return client.FindTraceByID(funcCtx, req)
		}
		partialTraces, err := q.forLiveStoreRing(ctx, forEach)
		if err != nil {
			return nil, fmt.Errorf("error querying live-stores in Querier.FindTraceByID: %w", err)
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
				return nil, fmt.Errorf("error combining live-store results in Querier.FindTraceByID: %w", err)
			}

			spanCountTotal += int64(spanCount)
			traceCountTotal++
			if resp.Metrics != nil {
				inspectedBytes += resp.Metrics.InspectedBytes
			}
		}

		span.AddEvent("done searching live-stores", oteltrace.WithAttributes(
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
		opts.RF1After = req.RF1After

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

	if q.externalClient != nil {
		if req.QueryMode == QueryModeExternal || req.QueryMode == QueryModeAll {
			span.AddEvent("searching external", oteltrace.WithAttributes(
				attribute.String("traceID", hex.EncodeToString(req.TraceID)),
				attribute.Int64("timeStart", timeStart),
				attribute.Int64("timeEnd", timeEnd),
			))
			externalResp, err := q.externalClient.TraceByID(ctx, userID, req.TraceID, timeStart, timeEnd)
			if err != nil {
				retErr := fmt.Errorf("error querying external in Querier.FindTraceByID: %w", err)
				span.RecordError(retErr)
				return nil, retErr
			}
			span.AddEvent("done searching external", oteltrace.WithAttributes(
				attribute.Int("spansFound", countSpans(externalResp.Trace)),
			))
			_, err = combiner.Consume(externalResp.Trace)
			if err != nil {
				return nil, err
			}
		}
	} else if req.QueryMode == QueryModeExternal {
		return nil, fmt.Errorf("external mode is not enabled")
	}

	completeTrace, _ := combiner.Result()
	resp := &tempopb.TraceByIDResponse{
		Trace:   completeTrace,
		Metrics: &tempopb.TraceByIDMetrics{InspectedBytes: inspectedBytes},
	}

	if combiner.IsPartialTrace() {
		resp.Status = tempopb.PartialStatus_PARTIAL
		resp.Message = fmt.Sprintf("Trace exceeds maximum size of %d bytes, a partial trace is returned", maxBytes)
	}

	return resp, nil
}

// forLiveStoreRing runs f, in parallel, for instances selected from the live-store ring.
func (q *Querier) forLiveStoreRing(ctx context.Context, f forEachFn) ([]any, error) {
	if ctx.Err() != nil {
		_ = level.Debug(log.Logger).Log("forLiveStoreRing context error", "ctx.Err()", ctx.Err().Error())
		return nil, ctx.Err()
	}

	ctx, span := tracer.Start(ctx, "Querier.forLiveStoreRing")
	defer span.End()

	if q.partitionRing == nil {
		return nil, errors.New("forLiveStoreRing: partition ring is not configured")
	}

	rs, err := q.partitionRing.GetReplicationSetsForOperation(ring.Read)
	if err != nil {
		return nil, fmt.Errorf("error finding partition ring replicas: %w", err)
	}
	return forPartitionRingReplicaSets(ctx, q, rs, f)
}

// forLiveStoreMetricsRing runs f, in parallel, for instances selected from the live-store ring.
func (q *Querier) forLiveStoreMetricsRing(ctx context.Context, f forEachMetricsFn) ([]any, error) {
	if ctx.Err() != nil {
		_ = level.Debug(log.Logger).Log("forLiveStoreMetricsRing context error", "ctx.Err()", ctx.Err().Error())
		return nil, ctx.Err()
	}

	ctx, span := tracer.Start(ctx, "Querier.forLiveStoreMetricsRing")
	defer span.End()

	if q.partitionRing == nil {
		return nil, errors.New("forLiveStoreMetricsRing: partition ring is not configured")
	}
	rs, err := q.partitionRing.GetReplicationSetsForOperation(ring.Read)
	if err != nil {
		return nil, fmt.Errorf("error finding partition ring replicas: %w", err)
	}
	return forPartitionRingReplicaSets(ctx, q, rs, f)
}

func (q *Querier) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	if _, err := validation.ExtractValidTenantID(ctx); err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchRecent: %w", err)
	}

	results, err := q.forLiveStoreRing(ctx, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchRecent(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying live-stores in Querier.SearchRecent: %w", err)
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
	userID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTags: %w", err)
	}

	maxDataSize := q.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewDistinctString(maxDataSize, req.MaxTagsPerScope, req.StaleValuesThreshold)
	var inspectedBytes uint64

	results, err := q.forLiveStoreRing(ctx, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTags(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying live-stores in Querier.SearchTags: %w", err)
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
	orgID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.SearchTags: %w", err)
	}

	maxBytesPerTag := q.limits.MaxBytesPerTagValuesQuery(orgID)
	distinctValues := collector.NewScopedDistinctString(maxBytesPerTag, req.MaxTagsPerScope, req.StaleValuesThreshold)
	var inspectedBytes uint64

	// Get results from all live stores.
	results, err := q.forLiveStoreRing(ctx, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTagsV2(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying live-stores in Querier.SearchTags: %w", err)
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
	userID, err := validation.ExtractValidTenantID(ctx)
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

	results, err := q.forLiveStoreRing(ctx, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTagValues(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying live-stores in Querier.SearchTagValues: %w", err)
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
	userID, err := validation.ExtractValidTenantID(ctx)
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

	results, err := q.forLiveStoreRing(ctx, func(ctx context.Context, client tempopb.QuerierClient) (any, error) {
		return client.SearchTagValuesV2(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying live-stores in Querier.SearchTagValues: %w", err)
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
	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
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
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
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

		return q.engine.ExecuteSearch(ctx, req.SearchReq, fetcher, q.limits.UnsafeQueryHints(tenantID))
	}

	return q.store.Search(ctx, meta, req.SearchReq, opts)
}

func (q *Querier) internalTagsSearchBlockV2(ctx context.Context, req *tempopb.SearchTagsBlockRequest) (*tempopb.SearchTagsV2Response, error) {
	// For the intrinsic scope there is nothing to do in the querier,
	// these are always added by the frontend.
	if req.SearchReq.Scope == api.ParamScopeIntrinsic {
		return &tempopb.SearchTagsV2Response{}, nil
	}

	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
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
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
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
	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return &tempopb.SearchTagValuesResponse{}, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
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
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
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
	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return &tempopb.SearchTagValuesV2Response{}, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
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
		Size_:            req.Size_,
		IndexPageSize:    req.IndexPageSize,
		TotalRecords:     req.TotalRecords,
		BlockID:          blockID,
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

func countSpans(trace *tempopb.Trace) int {
	count := 0
	for _, b := range trace.ResourceSpans {
		for _, ss := range b.ScopeSpans {
			count += len(ss.Spans)
		}
	}
	return count
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
