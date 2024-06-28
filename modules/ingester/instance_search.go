package ingester

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	ot_log "github.com/opentracing/opentracing-go/log"
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
	"github.com/opentracing/opentracing-go"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.Search")
	defer span.Finish()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	maxResults := int(req.Limit)
	// if limit is not set, use a safe default
	if maxResults == 0 {
		maxResults = 20
	}

	span.LogFields(ot_log.String("SearchRequest", req.String()))

	var (
		resultsMtx = sync.Mutex{}
		combiner   = traceql.NewMetadataCombiner()
		metrics    = &tempopb.SearchMetrics{}
		opts       = common.DefaultSearchOptions()
		anyErr     atomic.Error
	)

	search := func(blockID uuid.UUID, block common.Searcher, spanName string) {
		span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchBlock."+spanName)
		defer span.Finish()

		span.LogFields(ot_log.Event("block entry mtx acquired"))
		span.SetTag("blockID", blockID)

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
	span.LogFields(ot_log.String("msg", "acquired headblock mtx"))
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
	span.LogFields(ot_log.String("msg", "acquired blocks mtx"))

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
	v2Response, err := i.SearchTagsV2(ctx, scope)
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
func (i *instance) SearchTagsV2(ctx context.Context, scope string) (*tempopb.SearchTagsV2Response, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.SearchTagsV2")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

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

	searchBlock := func(ctx context.Context, s common.Searcher, spanName string) error {
		span, ctx := opentracing.StartSpanFromContext(ctx, "instance.SearchTags."+spanName)
		defer span.Finish()

		if s == nil {
			return nil
		}
		if distinctValues.Exceeded() {
			return nil
		}
		err = s.SearchTags(ctx, attributeScope, func(t string, scope traceql.AttributeScope) {
			distinctValues.Collect(scope.String(), t)
		}, common.DefaultSearchOptions())
		if err != nil && !errors.Is(err, common.ErrUnsupported) {
			return fmt.Errorf("unexpected error searching tags: %w", err)
		}

		return nil
	}

	i.headBlockMtx.RLock()
	span.LogFields(ot_log.String("msg", "acquired headblock mtx"))
	err = searchBlock(ctx, i.headBlock, "headBlock")
	i.headBlockMtx.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()
	span.LogFields(ot_log.String("msg", "acquired blocks mtx"))

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

	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.SearchTagValuesV2")
	defer span.Finish()

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	valueCollector := collector.NewDistinctValue[tempopb.TagValue](limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

	engine := traceql.NewEngine()

	wg := boundedwaitgroup.New(20) // TODO: Make configurable
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

	query := traceql.ExtractMatchers(req.Query)

	searchBlock := func(ctx context.Context, s common.Searcher) error {
		if anyErr.Load() != nil {
			return nil // Early exit if any error has occurred
		}

		if maxBlocks > 0 && inspectedBlocks.Inc() > maxBlocks {
			return nil
		}

		// if the query is empty or autocomplete filtering is disabled, use the old search
		if traceql.IsEmptyQuery(query) {
			return s.SearchTagValuesV2(ctx, tag, traceql.MakeCollectTagValueFunc(valueCollector.Collect), common.DefaultSearchOptions())
		}

		// otherwise use the filtered search
		fetcher := traceql.NewTagValuesFetcherWrapper(func(ctx context.Context, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback) error {
			return s.FetchTagValues(ctx, req, cb, common.DefaultSearchOptions())
		})

		return engine.ExecuteTagValues(ctx, tag, query, traceql.MakeCollectTagValueFunc(valueCollector.Collect), fetcher)
	}

	// head block
	// A warning about deadlocks!!  This area does a hard-acquire of both mutexes.
	// To avoid deadlocks this function and all others must acquire them in
	// the ** same_order ** or else!!! i.e. another function can't acquire blocksMtx
	// then headblockMtx. Even if the likelihood is low it is a statistical certainly
	// that eventually a deadlock will occur.
	i.headBlockMtx.RLock()
	span.LogFields(ot_log.String("msg", "acquired headblock mtx"))
	if i.headBlock != nil {
		wg.Add(1)
		go func() {
			span, ctx := opentracing.StartSpanFromContext(ctx, "instance.SearchTagValuesV2.headBlock")
			defer span.Finish()
			defer i.headBlockMtx.RUnlock()
			defer wg.Done()
			if err := searchBlock(ctx, i.headBlock); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err))
			}
		}()
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()
	span.LogFields(ot_log.String("msg", "acquired blocks mtx"))

	// completed blocks
	for _, b := range i.completeBlocks {
		wg.Add(1)
		go func(b *LocalBlock) {
			span, ctx := opentracing.StartSpanFromContext(ctx, "instance.SearchTagValuesV2.completedBlock")
			defer span.Finish()
			defer wg.Done()
			if err := searchBlock(ctx, b); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		wg.Add(1)
		go func(b common.WALBlock) {
			span, ctx := opentracing.StartSpanFromContext(ctx, "instance.SearchTagValuesV2.completingBlock")
			defer span.Finish()
			defer wg.Done()
			if err := searchBlock(ctx, b); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	wg.Wait()

	if err := anyErr.Load(); err != nil {
		return nil, err
	}

	if valueCollector.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "userID", userID, "limit", limit, "total", valueCollector.TotalDataSize())
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
