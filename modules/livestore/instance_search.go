/*
* livestore is based on ingester/instance_search.go any changes here should be reflected there and vice versa.
 */
package livestore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/segmentio/fasthash/fnv1a"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/collector"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var (
	tracer = otel.Tracer("modules/livestore")

	errComplete = errors.New("complete")
)

const (
	timeBuffer = 5 * time.Minute
)

type block interface {
	common.Searcher
	common.Finder
}

// blockFn defines a function that processes a single block
type blockFn func(ctx context.Context, meta *backend.BlockMeta, b block) error

// iterateBlocks provides a way to iterate over all blocks (head, wal, complete)
// using concurrent processing with bounded concurrency.
func (i *instance) iterateBlocks(ctx context.Context, reqStart, reqEnd time.Time, fn blockFn) error {
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	var anyErr atomic.Error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	handleErr := func(err error) {
		if err == nil {
			return
		}
		cancel()

		// we're not storing errComplete for obvious reasons. context.Canceled is ignored b/c it may be due to the
		// cancel above which is not an error. if the context is cancelled, by something above this method than the caller
		// can still detect and return it
		if errors.Is(err, errComplete) || errors.Is(err, context.Canceled) {
			return
		}
		anyErr.Store(err)
	}

	if i.headBlock != nil {
		meta := i.headBlock.BlockMeta()
		if includeBlock(meta, reqStart, reqEnd) {
			ctx, span := tracer.Start(ctx, "process.headBlock")
			span.SetAttributes(attribute.String("blockID", meta.BlockID.String()))

			if err := fn(ctx, meta, i.headBlock); err != nil {
				handleErr(fmt.Errorf("processing head block (%s): %w", meta.BlockID, err))
			}
			span.End()
		}
	}

	if err := anyErr.Load(); err != nil {
		return err
	}

	wg := boundedwaitgroup.New(i.Cfg.QueryBlockConcurrency)

	// Process wal blocks
	for _, b := range i.walBlocks {
		if ctx.Err() != nil {
			continue
		}

		meta := b.BlockMeta()
		if !includeBlock(meta, reqStart, reqEnd) {
			continue
		}

		wg.Add(1)
		go func(block common.WALBlock) {
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}

			ctx, span := tracer.Start(ctx, "process.walBlock")
			span.SetAttributes(attribute.String("blockID", meta.BlockID.String()))
			defer span.End()

			if err := fn(ctx, meta, block); err != nil {
				handleErr(fmt.Errorf("processing wal block (%s): %w", meta.BlockID, err))
			}
		}(b)
	}

	// Process complete blocks
	for _, b := range i.completeBlocks {
		if ctx.Err() != nil {
			continue
		}

		meta := b.BlockMeta()
		if !includeBlock(meta, reqStart, reqEnd) {
			continue
		}

		wg.Add(1)
		go func(block *ingester.LocalBlock) {
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}

			ctx, span := tracer.Start(ctx, "process.completeBlock")
			span.SetAttributes(attribute.String("blockID", meta.BlockID.String()))
			defer span.End()

			if err := fn(ctx, meta, block); err != nil {
				handleErr(fmt.Errorf("processing complete block (%s): %w", meta.BlockID, err))
			}
		}(b)
	}

	wg.Wait()

	if err := anyErr.Load(); err != nil {
		return err
	}

	return nil
}

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	ctx, span := tracer.Start(ctx, "instance.Search")
	defer span.End()

	maxResults := int(req.Limit)
	// if limit is not set, use a safe default
	if maxResults == 0 {
		maxResults = 20
	}

	span.AddEvent("SearchRequest", oteltrace.WithAttributes(attribute.String("request", req.String())))

	mostRecent := false
	if len(req.Query) > 0 {
		rootExpr, err := traceql.Parse(req.Query)
		if err != nil {
			return nil, fmt.Errorf("error parsing query: %w", err)
		}

		ok := false
		if mostRecent, ok = rootExpr.Hints.GetBool(traceql.HintMostRecent, false); !ok {
			mostRecent = false
		}
	}

	var (
		resultsMtx = sync.Mutex{}
		combiner   = traceql.NewMetadataCombiner(maxResults, mostRecent)
		metrics    = &tempopb.SearchMetrics{}
		opts       = common.DefaultSearchOptions()
	)

	search := func(ctx context.Context, blockMeta *backend.BlockMeta, b block) error {
		var resp *tempopb.SearchResponse
		var err error

		// if the combiner is complete for the block's end time, we can skip searching it
		if combiner.IsCompleteFor(uint32(blockMeta.EndTime.Unix())) {
			return errComplete
		}

		if api.IsTraceQLQuery(req) {
			// note: we are creating new engine for each wal block,
			// and engine.ExecuteSearch is parsing the query for each block
			resp, err = traceql.NewEngine().ExecuteSearch(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return b.Fetch(ctx, req, opts)
			}))
		} else {
			resp, err = b.Search(ctx, req, opts)
		}

		if errors.Is(err, util.ErrUnsupported) {
			level.Warn(i.logger).Log("msg", "block does not support search", "blockID", blockMeta.BlockID)
			return nil
		}
		if err != nil {
			return err
		}

		if resp == nil {
			return nil
		}

		resultsMtx.Lock()
		defer resultsMtx.Unlock()

		if resp.Metrics != nil {
			metrics.InspectedTraces += resp.Metrics.InspectedTraces
			metrics.InspectedBytes += resp.Metrics.InspectedBytes
		}

		for _, tr := range resp.Traces {
			combiner.AddMetadata(tr)
			if combiner.IsCompleteFor(traceql.TimestampNever) {
				return errComplete
			}
		}

		return nil
	}

	err := i.iterateBlocks(ctx, time.Unix(int64(req.Start), 0), time.Unix(int64(req.End), 0), search)
	if err != nil {
		level.Error(i.logger).Log("msg", "error in Search", "err", err)
		return nil, fmt.Errorf("Search: %w", err)
	}

	return &tempopb.SearchResponse{
		Traces:  combiner.Metadata(),
		Metrics: metrics,
	}, nil
}

func (i *instance) SearchTags(ctx context.Context, scope string) (*tempopb.SearchTagsResponse, error) {
	v2Response, err := i.SearchTagsV2(ctx, &tempopb.SearchTagsRequest{Scope: scope})
	if err != nil {
		return nil, err
	}

	distinctValues := collector.NewDistinctString(0, 0, 0) // search tags v2 enforces the limit

	// flatten v2 response
	for _, s := range v2Response.Scopes {
		for _, t := range s.Tags {
			distinctValues.Collect(t)
		}
	}

	return &tempopb.SearchTagsResponse{
		TagNames: distinctValues.Strings(),
		Metrics:  v2Response.Metrics, // send metrics with response
	}, nil
}

// SearchTagsV2 calls SearchTags for each scope and returns the results.
func (i *instance) SearchTagsV2(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsV2Response, error) {
	ctx, span := tracer.Start(ctx, "instance.SearchTagsV2")
	defer span.End()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	scope := req.Scope

	if scope == api.ParamScopeIntrinsic {
		// For the intrinsic scope there is nothing to do in the live store,
		// these are always added by the frontend.
		return &tempopb.SearchTagsV2Response{}, nil
	}

	// parse for normal scopes
	attributeScope := traceql.AttributeScopeFromString(scope)
	if attributeScope == traceql.AttributeScopeUnknown {
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	maxBytestPerTags := i.overrides.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewScopedDistinctString(maxBytestPerTags, req.MaxTagsPerScope, req.StaleValuesThreshold)
	mc := collector.NewMetricsCollector()

	engine := traceql.NewEngine()
	query := traceql.ExtractMatchers(req.Query)

	searchBlock := func(ctx context.Context, _ *backend.BlockMeta, b block) error {
		if b == nil {
			return nil
		}

		if distinctValues.Exceeded() {
			return errComplete
		}

		// if the query is empty, use the old search
		if traceql.IsEmptyQuery(query) {
			err = b.SearchTags(ctx, attributeScope, func(t string, scope traceql.AttributeScope) {
				distinctValues.Collect(scope.String(), t)
			}, mc.Add, common.DefaultSearchOptions())

			if err != nil && !errors.Is(err, util.ErrUnsupported) {
				return err
			}

			return nil
		}

		// otherwise use the filtered search
		fetcher := traceql.NewTagNamesFetcherWrapper(func(ctx context.Context, req traceql.FetchTagsRequest, cb traceql.FetchTagsCallback) error {
			return b.FetchTagNames(ctx, req, cb, mc.Add, common.DefaultSearchOptions())
		})

		return engine.ExecuteTagNames(ctx, attributeScope, query, func(tag string, scope traceql.AttributeScope) bool {
			return distinctValues.Collect(scope.String(), tag)
		}, fetcher)
	}

	err = i.iterateBlocks(ctx, time.Unix(int64(req.Start), 0), time.Unix(int64(req.End), 0), searchBlock)
	if err != nil {
		level.Error(i.logger).Log("msg", "error in SearchTagsV2", "err", err)
		return nil, fmt.Errorf("SearchTagsV2: %w", err)
	}

	if distinctValues.Exceeded() {
		level.Warn(i.logger).Log("msg", "Search of tags exceeded limit, reduce cardinality or size of tags", "orgID", userID, "stopReason", distinctValues.StopReason())
	}

	collected := distinctValues.Strings()
	resp := &tempopb.SearchTagsV2Response{
		Scopes: make([]*tempopb.SearchTagsV2Scope, 0, len(collected)+1), // +1 for intrinsic below
		Metrics: &tempopb.MetadataMetrics{
			InspectedBytes: mc.TotalValue(), // capture metrics
		},
	}
	for scope, vals := range collected {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scope,
			Tags: vals,
		})
	}

	return resp, nil
}

func (i *instance) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	tagName := req.TagName
	limit := req.MaxTagValues
	staleValueThreshold := req.StaleValueThreshold

	maxBytesPerTagValues := i.overrides.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewDistinctString(maxBytesPerTagValues, limit, staleValueThreshold)
	mc := collector.NewMetricsCollector()

	var inspectedBlocks, maxBlocks int
	if limit := i.overrides.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = limit
	}

	search := func(ctx context.Context, _ *backend.BlockMeta, b block) error {
		if maxBlocks > 0 && inspectedBlocks >= maxBlocks {
			return nil
		}

		if b == nil {
			return nil
		}

		if distinctValues.Exceeded() {
			return errComplete
		}

		inspectedBlocks++
		err = b.SearchTagValues(ctx, tagName, distinctValues.Collect, mc.Add, common.DefaultSearchOptions())
		if err != nil && !errors.Is(err, util.ErrUnsupported) {
			return fmt.Errorf("unexpected error searching tag values (%s): %w", tagName, err)
		}

		return nil
	}

	err = i.iterateBlocks(ctx, time.Unix(int64(req.Start), 0), time.Unix(int64(req.End), 0), search)
	if err != nil {
		level.Error(i.logger).Log("msg", "error in SearchTagValues", "err", err)
		return nil, fmt.Errorf("SearchTagValues: %w", err)
	}

	if distinctValues.Exceeded() {
		level.Warn(i.logger).Log("msg", "Search of tags exceeded limit,  reduce cardinality or size of tags", "tag", tagName, "orgID", userID, "stopReason", distinctValues.StopReason())
	}

	return &tempopb.SearchTagValuesResponse{
		TagValues: distinctValues.Strings(),
		Metrics:   &tempopb.MetadataMetrics{InspectedBytes: mc.TotalValue()},
	}, nil
}

func (i *instance) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	ctx, span := tracer.Start(ctx, "instance.SearchTagValuesV2")
	defer span.End()

	limit := i.overrides.MaxBytesPerTagValuesQuery(userID)
	vCollector := collector.NewDistinctValue(limit, req.MaxTagValues, req.StaleValueThreshold, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
	mCollector := collector.NewMetricsCollector() // to collect bytesRead metric

	engine := traceql.NewEngine()

	var inspectedBlocks atomic.Int32
	var maxBlocks int32
	if limit := i.overrides.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = int32(limit)
	}

	tag, err := traceql.ParseIdentifier(req.TagName)
	if err != nil {
		return nil, err
	}
	if tag == traceql.IntrinsicLinkTraceIDAttribute ||
		tag == traceql.IntrinsicLinkSpanIDAttribute ||
		tag == traceql.IntrinsicSpanIDAttribute ||
		tag == traceql.IntrinsicTraceIDAttribute ||
		tag == traceql.IntrinsicParentIDAttribute {
		// do not return tag values for IDs
		return &tempopb.SearchTagValuesV2Response{}, nil
	}

	query := traceql.ExtractMatchers(req.Query)
	// cacheKey will be same for all blocks in a request so only compute it once
	// NOTE: cacheKey tag name and query, so if we start respecting start and end, add them to the cacheKey
	cacheKey := searchTagValuesV2CacheKey(req, limit, "cache_search_tagvaluesv2")

	// helper functions as closures, to access local variables
	search := func(ctx context.Context, s common.Searcher) error {
		// note the interaction below with searchWithCache. if we ever return errComplete for reasons besides this we may need to adjust the error handling there
		if maxBlocks > 0 && inspectedBlocks.Inc() > maxBlocks {
			return errComplete
		}

		if traceql.IsEmptyQuery(query) {
			return s.SearchTagValuesV2(ctx, tag, traceql.MakeCollectTagValueFunc(vCollector.Collect), mCollector.Add, common.DefaultSearchOptions())
		}

		// Otherwise, use the filtered search
		fetcher := traceql.NewTagValuesFetcherWrapper(func(ctx context.Context, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback) error {
			return s.FetchTagValues(ctx, req, cb, mCollector.Add, common.DefaultSearchOptions())
		})

		return engine.ExecuteTagValues(ctx, tag, query, traceql.MakeCollectTagValueFunc(vCollector.Collect), fetcher)
	}

	searchWithCache := func(ctx context.Context, _ *backend.BlockMeta, b block) error {
		// if not a local block, fall back to regular search
		localB, ok := b.(*ingester.LocalBlock)
		if !ok {
			return search(ctx, b)
		}

		// pulled from context to add attrs below
		span := oteltrace.SpanFromContext(ctx)

		// check the cache first
		cacheData, err := localB.GetDiskCache(ctx, cacheKey)
		if err != nil {
			// just log the error and move on...we will search the block
			_ = level.Warn(i.logger).Log("msg", "GetDiskCache failed", "err", err)
			return nil
		}

		// we got data...unmarshall, and add values to central collector and add bytesRead
		if len(cacheData) > 0 {
			resp := &tempopb.SearchTagValuesV2Response{}
			err = proto.Unmarshal(cacheData, resp)
			if err != nil {
				_ = level.Warn(i.logger).Log("msg", "GetDiskCache unmarshal failed", "err", err)
				return nil
			}

			span.SetAttributes(attribute.Bool("cached", true))
			// Instead of the reporting the InspectedBytes of the cached response.
			// we report the size of cacheData as the Inspected bytes in case we hit disk cache.
			// we do this because, because it's incorrect and misleading to report the metrics of cachedResponse
			// we report the size of the cacheData as the amount of data was read to search this block.
			// this can skew our metrics because this will be lower than the data read to search the block.
			// we can remove this if this becomes an issue but leave it in for now to more accurate.
			mCollector.Add(uint64(len(cacheData)))

			for _, v := range resp.TagValues {
				if vCollector.Collect(*v) {
					return errComplete
				}
			}
			return nil
		}

		// cache miss, search the block. We will cache the results if we find any.
		span.SetAttributes(attribute.Bool("cached", false))
		// using local collector to collect values from the block and cache them.
		localCol := collector.NewDistinctValue[tempopb.TagValue](limit, req.MaxTagValues, req.StaleValueThreshold, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

		if err := search(ctx, localB); err != nil { // note that errComplete could be returned here but it's ok to pass it up b/c it means no work was done and the localCol is invalid
			return err
		}

		// marshal the values local collector and set the cache
		values := localCol.Values()
		v2RespProto, err := valuesToTagValuesV2RespProto(values)
		if err == nil && len(v2RespProto) > 0 {
			err2 := localB.SetDiskCache(ctx, cacheKey, v2RespProto)
			if err2 != nil {
				_ = level.Warn(i.logger).Log("msg", "SetDiskCache failed", "err", err2)
			}
		}

		// now add values to the central collector to make sure they are included in the response.
		for _, v := range values {
			if vCollector.Collect(v) {
				return errComplete
			}
		}
		return nil
	}

	err = i.iterateBlocks(ctx, time.Unix(int64(req.Start), 0), time.Unix(int64(req.End), 0), searchWithCache)
	if err != nil {
		level.Error(i.logger).Log("msg", "error in SearchTagValuesV2", "err", err)
		return nil, fmt.Errorf("SearchTagValuesV2: %w", err)
	}

	if vCollector.Exceeded() {
		_ = level.Warn(i.logger).Log("msg", "size of tag values exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "tenant", userID, "limit", limit, "size", vCollector.Size())
	}

	resp := &tempopb.SearchTagValuesV2Response{
		Metrics: &tempopb.MetadataMetrics{InspectedBytes: mCollector.TotalValue()}, // include metrics in response
	}

	for _, v := range vCollector.Values() {
		v2 := v
		resp.TagValues = append(resp.TagValues, &v2)
	}

	return resp, nil
}

func (i *instance) FindByTraceID(ctx context.Context, traceID []byte) (*tempopb.TraceByIDResponse, error) {
	var (
		metricsMtx sync.Mutex
		metrics    = tempopb.TraceByIDMetrics{}
		combiner   = trace.NewCombiner(math.MaxInt64, true)
	)

	// Check live traces first
	i.liveTracesMtx.Lock()
	if liveTrace, ok := i.liveTraces.Traces[util.HashForTraceID(traceID)]; ok {
		tempTrace := &tempopb.Trace{}
		tempTrace.ResourceSpans = liveTrace.Batches
		// Previously there was some logic here to add inspected bytes in the ingester. But its hard to do with the different
		// live traces format and feels inaccurate.
		_, err := combiner.Consume(tempTrace)
		if err != nil {
			i.liveTracesMtx.Unlock()
			return nil, fmt.Errorf("unable to unmarshal liveTrace: %w", err)
		}
	}
	i.liveTracesMtx.Unlock()

	search := func(ctx context.Context, _ *backend.BlockMeta, b block) error {
		trace, err := b.FindTraceByID(ctx, traceID, common.DefaultSearchOptions())
		if err != nil {
			return err
		}
		if trace == nil {
			return nil
		}

		_, err = combiner.Consume(trace.Trace)
		if err != nil {
			return err
		}

		if trace.Metrics != nil {
			metricsMtx.Lock()
			defer metricsMtx.Unlock()

			metrics.InspectedBytes += trace.Metrics.InspectedBytes
		}
		return nil
	}

	err := i.iterateBlocks(ctx, time.Unix(0, 0), time.Unix(0, 0), search)
	if err != nil {
		level.Error(i.logger).Log("msg", "error in FindTraceByID", "err", err)
		return nil, fmt.Errorf("error searching for trace: %w", err)
	}

	result, _ := combiner.Result()
	response := &tempopb.TraceByIDResponse{
		Trace:   result,
		Metrics: &metrics,
	}
	return response, nil
}

// QueryRange returns metrics.
func (i *instance) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	e := traceql.NewEngine()

	// Compile the raw version of the query for head and wal blocks
	// These aren't cached and we put them all into the same evaluator
	// for efficiency.
	// TODO MRD look into how to propagate unsafe query hints.
	rawEval, err := e.CompileMetricsQueryRange(req, int(req.Exemplars), i.Cfg.Metrics.TimeOverlapCutoff, false)
	if err != nil {
		return nil, err
	}

	// This is a summation version of the query for complete blocks
	// which can be cached. They are timeseries, so they need the job-level evaluator.
	jobEval, err := traceql.NewEngine().CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-i.Cfg.CompleteBlockTimeout).Add(-timeBuffer)
	if req.Start < uint64(cutoff.UnixNano()) {
		return nil, fmt.Errorf("time range must be within last %v", i.Cfg.CompleteBlockTimeout)
	}

	expr, err := traceql.Parse(req.Query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	unsafe := i.overrides.UnsafeQueryHints(i.tenantID)

	timeOverlapCutoff := i.Cfg.Metrics.TimeOverlapCutoff
	if v, ok := expr.Hints.GetFloat(traceql.HintTimeOverlapCutoff, unsafe); ok && v >= 0 && v <= 1.0 {
		timeOverlapCutoff = v
	}

	maxSeries := int(req.MaxSeries)
	maxSeriesReached := atomic.Bool{}
	maxSeriesReached.Store(false)

	search := func(ctx context.Context, _ *backend.BlockMeta, b block) error {
		if walBlock, ok := b.(common.WALBlock); ok {
			err := i.queryRangeWALBlock(ctx, walBlock, rawEval, maxSeries)
			if err != nil {
				return err
			}
			if maxSeries > 0 && rawEval.Length() > maxSeries {
				maxSeriesReached.Store(true)
				return errComplete
			}
			return nil
		}

		if localBlock, ok := b.(*ingester.LocalBlock); ok {
			resp, err := i.queryRangeCompleteBlock(ctx, localBlock, *req, timeOverlapCutoff, unsafe, int(req.Exemplars))
			if err != nil {
				return err
			}
			jobEval.ObserveSeries(resp)
			if maxSeries > 0 && jobEval.Length() > maxSeries {
				maxSeriesReached.Store(true)
				return errComplete
			}
			return nil
		}

		return fmt.Errorf("unexpected block type: %T", b)
	}

	err = i.iterateBlocks(ctx, time.Unix(0, int64(req.Start)), time.Unix(0, int64(req.End)), search)
	if err != nil {
		level.Error(i.logger).Log("msg", "error in QueryRange", "err", err)
		return nil, err
	}

	// Combine the raw results into the job results
	walResults := rawEval.Results().ToProto(req)
	jobEval.ObserveSeries(walResults)

	r := jobEval.Results()
	rr := r.ToProto(req)

	if maxSeriesReached.Load() {
		return &tempopb.QueryRangeResponse{
			Series: rr[:maxSeries],
			Status: tempopb.PartialStatus_PARTIAL,
		}, nil
	}

	return &tempopb.QueryRangeResponse{
		Series: rr,
	}, nil
}

func (i *instance) queryRangeWALBlock(ctx context.Context, b common.WALBlock, eval *traceql.MetricsEvaluator, maxSeries int) error {
	m := b.BlockMeta()
	ctx, span := tracer.Start(ctx, "instance.QueryRange.WALBlock", oteltrace.WithAttributes(
		attribute.String("block", m.BlockID.String()),
		attribute.Int64("blockSize", int64(m.Size_)),
	))
	defer span.End()

	fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return b.Fetch(ctx, req, common.DefaultSearchOptions())
	})

	return eval.Do(ctx, fetcher, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()), maxSeries)
}

func (i *instance) queryRangeCompleteBlock(ctx context.Context, b *ingester.LocalBlock, req tempopb.QueryRangeRequest, timeOverlapCutoff float64, unsafe bool, exemplars int) ([]*tempopb.TimeSeries, error) {
	m := b.BlockMeta()
	ctx, span := tracer.Start(ctx, "instance.QueryRange.CompleteBlock", oteltrace.WithAttributes(
		attribute.String("block", m.BlockID.String()),
		attribute.Int64("blockSize", int64(m.Size_)),
	))
	defer span.End()

	// Trim and align the request for this block. I.e. if the request is "Last Hour" we don't want to
	// cache the response for that, we want only the few minutes time range for this block. This has
	// size savings but the main thing is that the response is reuseable for any overlapping query.
	req.Start, req.End, req.Step = traceql.TrimToBlockOverlap(req.Start, req.End, req.Step, m.StartTime, m.EndTime)

	if req.Start >= req.End {
		// After alignment there is no overlap or something else isn't right
		return nil, nil
	}

	cached, name, err := i.queryRangeCacheGet(ctx, m, req)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(attribute.Bool("cached", cached != nil))

	if cached != nil {
		return cached.Series, nil
	}

	// Not in cache or not cacheable, so execute
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(&req, exemplars, timeOverlapCutoff, unsafe)
	if err != nil {
		return nil, err
	}
	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return b.Fetch(ctx, req, common.DefaultSearchOptions())
	})
	err = eval.Do(ctx, f, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()), int(req.MaxSeries))
	if err != nil {
		return nil, err
	}

	results := eval.Results().ToProto(&req)

	if name != "" {
		err = i.queryRangeCacheSet(ctx, m, name, &tempopb.QueryRangeResponse{
			Series: results,
		})
		if err != nil {
			return nil, fmt.Errorf("writing local query cache: %w", err)
		}
	}

	return results, nil
}

func (i *instance) queryRangeCacheGet(ctx context.Context, m *backend.BlockMeta, req tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, string, error) {
	hash := queryRangeHashForBlock(req)

	name := fmt.Sprintf("cache_query_range_%v.buf", hash)

	keyPath := backend.KeyPathForBlock((uuid.UUID)(m.BlockID), m.TenantID)
	reader, size, err := i.wal.LocalBackend().Read(ctx, name, keyPath, nil)
	if err != nil {
		if errors.Is(err, backend.ErrDoesNotExist) {
			// Not cached, but return the name/keypath so it can be set after
			return nil, name, nil
		}
		return nil, "", err
	}
	defer reader.Close()

	data, err := tempo_io.ReadAllWithEstimate(reader, size)
	if err != nil {
		return nil, "", err
	}

	resp := &tempopb.QueryRangeResponse{}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return nil, "", err
	}

	return resp, name, nil
}

func (i *instance) queryRangeCacheSet(ctx context.Context, m *backend.BlockMeta, name string, resp *tempopb.QueryRangeResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	keyPath := backend.KeyPathForBlock((uuid.UUID)(m.BlockID), m.TenantID)
	return i.wal.LocalBackend().Write(ctx, name, keyPath, bytes.NewReader(data), int64(len(data)), nil)
}

func queryRangeHashForBlock(req tempopb.QueryRangeRequest) uint64 {
	h := fnv1a.HashString64(req.Query)
	h = fnv1a.AddUint64(h, req.Start)
	h = fnv1a.AddUint64(h, req.End)
	h = fnv1a.AddUint64(h, req.Step)

	// TODO - caching for WAL blocks
	// Including trace count means we can safely cache results
	// for wal blocks which might receive new data
	// h = fnv1a.AddUint64(h, m.TotalObjects)

	return h
}

// includeBlock uses the provided time range to determine if the block should be included in the search.
func includeBlock(b *backend.BlockMeta, start, end time.Time) bool {
	startNano := start.UnixNano()
	endNano := end.UnixNano()

	if startNano == 0 || endNano == 0 {
		return true
	}

	blockStart := b.StartTime.UnixNano()
	blockEnd := b.EndTime.UnixNano()

	return blockStart <= endNano && blockEnd >= startNano
}

// searchTagValuesV2CacheKey generates a cache key for the searchTagValuesV2 request
// cache key is used as the filename to store the protobuf data on disk
func searchTagValuesV2CacheKey(req *tempopb.SearchTagValuesRequest, limit int, prefix string) string {
	var cacheKey string
	if req.Query != "" {
		q := traceql.ExtractMatchers(req.Query)
		if ast, err := traceql.Parse(q); err == nil {
			// forces the query into a canonical form
			cacheKey = ast.String()
		} else {
			// In case of a bad TraceQL query, we ignore the query and return unfiltered results.
			// if we fail to parse the query, we will assume query is empty and compute the cache key.
			cacheKey = ""
		}
	}

	// NOTE: we are not adding req.Start and req.End to the cache key because we don't respect the start and end
	// please add them to cacheKey if we start respecting them
	h := fnv1a.HashString64(req.TagName)
	h = fnv1a.AddString64(h, cacheKey)
	h = fnv1a.AddUint64(h, uint64(limit))

	return fmt.Sprintf("%s_%v.buf", prefix, h)
}

// valuesToTagValuesV2RespProto converts TagValues to a protobuf marshalled bytes
// this is slightly modified version of valuesToV2Response from querier.go
func valuesToTagValuesV2RespProto(tagValues []tempopb.TagValue) ([]byte, error) {
	// NOTE: we only cache TagValues and don't Marshal Metrics
	resp := &tempopb.SearchTagValuesV2Response{}
	resp.TagValues = make([]*tempopb.TagValue, 0, len(tagValues))

	for _, v := range tagValues {
		v2 := &v
		resp.TagValues = append(resp.TagValues, v2)
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return data, nil
}
