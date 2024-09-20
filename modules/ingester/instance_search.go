package ingester

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/segmentio/fasthash/fnv1a"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	ctx, span := tracer.Start(ctx, "instance.Search")
	defer span.End()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	maxResults := int(req.Limit)
	// if limit is not set, use a safe default
	if maxResults == 0 {
		maxResults = 20
	}

	span.AddEvent("SearchRequest", trace.WithAttributes(attribute.String("request", req.String())))

	var (
		resultsMtx = sync.Mutex{}
		combiner   = traceql.NewMetadataCombiner()
		metrics    = &tempopb.SearchMetrics{}
		opts       = common.DefaultSearchOptions()
		anyErr     atomic.Error
	)

	search := func(blockID uuid.UUID, block common.Searcher, spanName string) {
		ctx, span := tracer.Start(ctx, "instance.searchBlock."+spanName)
		defer span.End()

		span.AddEvent("block entry mtx acquired")
		span.SetAttributes(attribute.String("blockID", blockID.String()))

		var resp *tempopb.SearchResponse
		var err error

		if api.IsTraceQLQuery(req) {
			// note: we are creating new engine for each wal block,
			// and engine.ExecuteSearch is parsing the query for each block
			resp, err = traceql.NewEngine().ExecuteSearch(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return block.Fetch(ctx, req, opts)
			}))
		} else {
			resp, err = block.Search(ctx, req, opts)
		}

		if errors.Is(err, common.ErrUnsupported) {
			level.Warn(log.Logger).Log("msg", "block does not support search", "blockID", blockID)
			return
		}
		if errors.Is(err, context.Canceled) {
			// Ignore
			return
		}
		if err != nil {
			level.Error(log.Logger).Log("msg", "error searching block", "blockID", blockID, "err", err)
			anyErr.Store(err)
			return
		}

		if resp == nil {
			return
		}

		resultsMtx.Lock()
		defer resultsMtx.Unlock()

		if resp.Metrics != nil {
			metrics.InspectedTraces += resp.Metrics.InspectedTraces
			metrics.InspectedBytes += resp.Metrics.InspectedBytes
		}

		if combiner.Count() >= maxResults {
			return
		}

		for _, tr := range resp.Traces {
			combiner.AddMetadata(tr)
			if combiner.Count() >= maxResults {
				// Cancel all other tasks
				cancel()
				return
			}
		}
	}

	// Search headblock (synchronously)
	// Lock headblock separately from other blocks and release it as quickly as possible.
	// A warning about deadlocks!!  This area does a hard-acquire of both mutexes.
	// To avoid deadlocks this function and all others must acquire them in
	// the ** same_order ** or else!!! i.e. another function can't acquire blocksMtx
	// then headblockMtx. Even if the likelihood is low it is a statistical certainly
	// that eventually a deadlock will occur.
	i.headBlockMtx.RLock()
	span.AddEvent("acquired headblock mtx")
	if includeBlock(i.headBlock.BlockMeta(), req) {
		search(i.headBlock.BlockMeta().BlockID, i.headBlock, "headBlock")
	}
	i.headBlockMtx.RUnlock()
	if err := anyErr.Load(); err != nil {
		return nil, err
	}
	if combiner.Count() >= maxResults {
		return &tempopb.SearchResponse{
			Traces:  combiner.Metadata(),
			Metrics: metrics,
		}, nil
	}

	// Search all other blocks (concurrently)
	// Lock blocks mutex until all search tasks are finished and this function exits. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()
	span.AddEvent("acquired blocks mtx")

	wg := sync.WaitGroup{}

	for _, b := range i.completingBlocks {
		if !includeBlock(b.BlockMeta(), req) {
			continue
		}

		wg.Add(1)
		go func(b common.WALBlock) {
			defer wg.Done()
			search(b.BlockMeta().BlockID, b, "completingBlock")
		}(b)
	}

	for _, b := range i.completeBlocks {
		if !includeBlock(b.BlockMeta(), req) {
			continue
		}
		wg.Add(1)
		go func(b *LocalBlock) {
			defer wg.Done()
			search(b.BlockMeta().BlockID, b, "completeBlock")
		}(b)
	}

	wg.Wait()

	if err := anyErr.Load(); err != nil {
		return nil, err
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

	distinctValues := collector.NewDistinctString(0) // search tags v2 enforces the limit

	// flatten v2 response
	for _, s := range v2Response.Scopes {
		// SearchTags does not include intrinsics on an empty scope, but v2 does.
		if scope == "" && s.Name == api.ParamScopeIntrinsic {
			continue
		}

		for _, t := range s.Tags {
			distinctValues.Collect(t)
		}
	}

	return &tempopb.SearchTagsResponse{
		TagNames: distinctValues.Strings(),
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
	// check if it's the special intrinsic scope
	if scope == api.ParamScopeIntrinsic {
		return &tempopb.SearchTagsV2Response{
			Scopes: []*tempopb.SearchTagsV2Scope{
				{
					Name: api.ParamScopeIntrinsic,
					Tags: search.GetVirtualIntrinsicValues(),
				},
			},
		}, nil
	}

	// parse for normal scopes
	attributeScope := traceql.AttributeScopeFromString(scope)
	if attributeScope == traceql.AttributeScopeUnknown {
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewScopedDistinctString(limit)

	engine := traceql.NewEngine()
	query := traceql.ExtractMatchers(req.Query)

	searchBlock := func(ctx context.Context, s common.Searcher, spanName string) error {
		ctx, span := tracer.Start(ctx, "instance.SearchTagsV2."+spanName)
		defer span.End()

		if s == nil {
			return nil
		}
		if distinctValues.Exceeded() {
			return nil
		}

		// if the query is empty, use the old search
		if traceql.IsEmptyQuery(query) {
			err = s.SearchTags(ctx, attributeScope, func(t string, scope traceql.AttributeScope) {
				distinctValues.Collect(scope.String(), t)
			}, common.DefaultSearchOptions())
			if err != nil && !errors.Is(err, common.ErrUnsupported) {
				return fmt.Errorf("unexpected error searching tags: %w", err)
			}

			return nil
		}

		// otherwise use the filtered search
		fetcher := traceql.NewTagNamesFetcherWrapper(func(ctx context.Context, req traceql.FetchTagsRequest, cb traceql.FetchTagsCallback) error {
			return s.FetchTagNames(ctx, req, cb, common.DefaultSearchOptions())
		})

		return engine.ExecuteTagNames(ctx, attributeScope, query, func(tag string, scope traceql.AttributeScope) bool {
			distinctValues.Collect(scope.String(), tag)
			return distinctValues.Exceeded()
		}, fetcher)
	}

	i.headBlockMtx.RLock()
	span.AddEvent("acquired headblock mtx")
	err = searchBlock(ctx, i.headBlock, "headBlock")
	i.headBlockMtx.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()
	span.AddEvent("acquired blocks mtx")

	for _, b := range i.completingBlocks {
		if err = searchBlock(ctx, b, "completingBlock"); err != nil {
			return nil, fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}
	for _, b := range i.completeBlocks {
		if err = searchBlock(ctx, b, "completeBlock"); err != nil {
			return nil, fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tags in instance exceeded limit, reduce cardinality or size of tags", "userID", userID, "limit", limit)
	}

	collected := distinctValues.Strings()
	resp := &tempopb.SearchTagsV2Response{
		Scopes: make([]*tempopb.SearchTagsV2Scope, 0, len(collected)+1), // +1 for intrinsic below
	}
	for scope, vals := range collected {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scope,
			Tags: vals,
		})
	}

	// add intrinsic tags if scope is none
	if attributeScope == traceql.AttributeScopeNone {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: api.ParamScopeIntrinsic,
			Tags: search.GetVirtualIntrinsicValues(),
		})
	}

	return resp, nil
}

func (i *instance) SearchTagValues(ctx context.Context, tagName string) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := collector.NewDistinctString(limit)

	var inspectedBlocks, maxBlocks int
	if limit := i.limiter.limits.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = limit
	}

	search := func(s common.Searcher, dv *collector.DistinctString) error {
		if maxBlocks > 0 && inspectedBlocks >= maxBlocks {
			return nil
		}

		if s == nil {
			return nil
		}
		if dv.Exceeded() {
			return nil
		}

		inspectedBlocks++
		err = s.SearchTagValues(ctx, tagName, dv.Collect, common.DefaultSearchOptions())
		if err != nil && !errors.Is(err, common.ErrUnsupported) {
			return fmt.Errorf("unexpected error searching tag values (%s): %w", tagName, err)
		}

		return nil
	}

	i.headBlockMtx.RLock()
	err = search(i.headBlock, distinctValues)
	i.headBlockMtx.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	for _, b := range i.completingBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}
	for _, b := range i.completeBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", tagName, "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	return &tempopb.SearchTagValuesResponse{
		TagValues: distinctValues.Strings(),
	}, nil
}

func (i *instance) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	ctx, span := tracer.Start(ctx, "instance.SearchTagValuesV2")
	defer span.End()

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	valueCollector := collector.NewDistinctValue[tempopb.TagValue](limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

	engine := traceql.NewEngine()

	// we usually have 5-10 blocks on an ingester so cap of 20 is more than enough and usually more than the blocks
	// we need to search, and this also acts as the limit on the amount of search load on the ingester.
	wg := boundedwaitgroup.New(20)
	var anyErr atomic.Error
	var inspectedBlocks atomic.Int32
	var maxBlocks int32
	if limit := i.limiter.limits.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = int32(limit)
	}

	tag, err := traceql.ParseIdentifier(req.TagName)
	if err != nil {
		return nil, err
	}
	if tag == traceql.IntrinsicLinkTraceIDAttribute ||
		tag == traceql.IntrinsicLinkSpanIDAttribute ||
		tag == traceql.IntrinsicSpanIDAttribute ||
		tag == traceql.IntrinsicTraceIDAttribute {
		// do not return tag values for IDs
		return &tempopb.SearchTagValuesV2Response{}, nil
	}

	query := traceql.ExtractMatchers(req.Query)
	// cacheKey will be same for all blocks in a request so only compute it once
	// NOTE: cacheKey tag name and query, so if we start respecting start and end, add them to the cacheKey
	cacheKey := searchTagValuesV2CacheKey(req, limit, "cache_search_tagvaluesv2")

	// helper functions as closures, to access local variables
	performSearch := func(ctx context.Context, s common.Searcher, collector *collector.DistinctValue[tempopb.TagValue]) error {
		if traceql.IsEmptyQuery(query) {
			return s.SearchTagValuesV2(ctx, tag, traceql.MakeCollectTagValueFunc(collector.Collect), common.DefaultSearchOptions())
		}

		// Otherwise, use the filtered search
		fetcher := traceql.NewTagValuesFetcherWrapper(func(ctx context.Context, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback) error {
			return s.FetchTagValues(ctx, req, cb, common.DefaultSearchOptions())
		})

		return engine.ExecuteTagValues(ctx, tag, query, traceql.MakeCollectTagValueFunc(collector.Collect), fetcher)
	}

	exitEarly := func() bool {
		if anyErr.Load() != nil {
			return true // Early exit if any error has occurred
		}

		if maxBlocks > 0 && inspectedBlocks.Inc() > maxBlocks {
			return true
		}

		return false // Continue searching
	}

	searchBlock := func(ctx context.Context, s common.Searcher, spanName string) error {
		ctx, span := tracer.Start(ctx, "instance.SearchTagValuesV2."+spanName)
		defer span.End()

		if exitEarly() {
			return nil
		}

		return performSearch(ctx, s, valueCollector)
	}

	searchBlockWithCache := func(ctx context.Context, b *LocalBlock, spanName string) error {
		ctx, span := tracer.Start(ctx, "instance.SearchTagValuesV2."+spanName)
		defer span.End()

		if exitEarly() {
			return nil
		}

		// check the cache first
		cacheData, err := b.GetDiskCache(ctx, cacheKey)
		if err != nil {
			// just log the error and move on...we will search the block
			_ = level.Warn(log.Logger).Log("msg", "GetDiskCache failed", "err", err)
		}

		// we got data...unmarshall, and add values to central collector
		if len(cacheData) > 0 && err == nil {
			resp := &tempopb.SearchTagValuesV2Response{}
			err = proto.Unmarshal(cacheData, resp)
			if err != nil {
				return err
			}
			span.SetAttributes(attribute.Bool("cached", true))
			for _, v := range resp.TagValues {
				if valueCollector.Collect(*v) {
					break // we have reached the limit, so stop
				}
			}
			return nil
		}

		span.SetAttributes(attribute.Bool("cached", false))
		// results not in cache, so search the block
		// using a local collector to collect values from the block and set cache
		localCol := collector.NewDistinctValue[tempopb.TagValue](limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })
		localErr := performSearch(ctx, b, localCol)
		if localErr != nil {
			return localErr
		}

		// marshal the local collector and set the cache
		values := localCol.Values()
		valuesProto, err := valuesToTagValuesV2RespProto(values)
		if err == nil && len(valuesProto) > 0 {
			err2 := b.SetDiskCache(ctx, cacheKey, valuesProto)
			if err2 != nil {
				_ = level.Warn(log.Logger).Log("msg", "SetDiskCache failed", "err", err2)
			}
		}

		// add values to the central collector
		for _, v := range values {
			if valueCollector.Collect(v) {
				break // we have reached the limit, so stop
			}
		}
		return nil
	}

	// head block
	// A warning about deadlocks!!  This area does a hard-acquire of both mutexes.
	// To avoid deadlocks this function and all others must acquire them in
	// the ** same_order ** or else!!! i.e. another function can't acquire blocksMtx
	// then headblockMtx. Even if the likelihood is low it is a statistical certainly
	// that eventually a deadlock will occur.
	i.headBlockMtx.RLock()
	span.AddEvent("acquired headblock mtx")
	if i.headBlock != nil {
		wg.Add(1)
		go func() {
			defer i.headBlockMtx.RUnlock()
			defer wg.Done()
			if err := searchBlock(ctx, i.headBlock, "headBlock"); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err))
			}
		}()
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()
	span.AddEvent("acquired blocks mtx")

	// completed blocks
	for _, b := range i.completeBlocks {
		wg.Add(1)
		go func(b *LocalBlock) {
			defer wg.Done()
			if err := searchBlockWithCache(ctx, b, "completeBlocks"); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		wg.Add(1)
		go func(b common.WALBlock) {
			defer wg.Done()
			if err := searchBlock(ctx, b, "completingBlocks"); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	wg.Wait()

	if err := anyErr.Load(); err != nil {
		return nil, err
	}

	if valueCollector.Exceeded() {
		_ = level.Warn(log.Logger).Log("msg", "size of tag values exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "userID", userID, "limit", limit, "size", valueCollector.Size())
	}

	resp := &tempopb.SearchTagValuesV2Response{}

	for _, v := range valueCollector.Values() {
		v2 := v
		resp.TagValues = append(resp.TagValues, &v2)
	}

	return resp, nil
}

// includeBlock uses the provided time range to determine if the block should be included in the search.
func includeBlock(b *backend.BlockMeta, req *tempopb.SearchRequest) bool {
	start := int64(req.Start)
	end := int64(req.End)

	if start == 0 || end == 0 {
		return true
	}

	return b.StartTime.Unix() <= end && b.EndTime.Unix() >= start
}

func searchTagValuesV2CacheKey(req *tempopb.SearchTagValuesRequest, limit int, prefix string) string {
	query := req.Query
	if req.Query != "" {
		ast, err := traceql.Parse(req.Query)
		if err != nil { // this should never happen but in case it happens
			return ""
		}
		// forces the query into a canonical form
		query = ast.String()
	}

	// NOTE: we are not adding req.Start and req.End to the cache key because we don't respect the start and end
	// please add them to cacheKey if we start respecting them
	h := fnv1a.HashString64(req.TagName)
	h = fnv1a.AddString64(h, query)
	h = fnv1a.AddUint64(h, uint64(limit))

	return fmt.Sprintf("%s_%v.buf", prefix, h)
}

// valuesToTagValuesV2RespProto converts TagValues to a protobuf marshalled bytes
// this is slightly modified version of valuesToV2Response from querier.go
func valuesToTagValuesV2RespProto(tagValues []tempopb.TagValue) ([]byte, error) {
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
