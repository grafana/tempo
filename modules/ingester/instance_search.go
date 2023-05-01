package ingester

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/user"
	"go.uber.org/atomic"
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

	sr := search.NewResults()
	defer sr.Close() // signal all running workers to quit

	// Lock blocks mutex until all search tasks have been created. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	i.searchWAL(ctx, req, sr)
	i.searchLocalBlocks(ctx, req, sr)
	i.blocksMtx.RUnlock()

	sr.AllWorkersStarted()

	// read and combine search results
	resultsMap := map[string]*tempopb.TraceSearchMetadata{}

	// collect results from all the goroutines via sr.Results channel.
	// range loop will exit when sr.Results channel is closed.
	for result := range sr.Results() {
		// exit early and Propagate error upstream
		if sr.Error() != nil {
			return nil, sr.Error()
		}

		// Dedupe/combine results
		if existing := resultsMap[result.TraceID]; existing != nil {
			search.CombineSearchResults(existing, result)
		} else {
			resultsMap[result.TraceID] = result
		}

		if len(resultsMap) >= maxResults {
			sr.Close() // signal pending workers to exit
			break
		}
	}

	// can happen when we have only error, and no results
	if sr.Error() != nil {
		return nil, sr.Error()
	}

	results := make([]*tempopb.TraceSearchMetadata, 0, len(resultsMap))
	for _, result := range resultsMap {
		results = append(results, result)
	}

	// Sort
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartTimeUnixNano > results[j].StartTimeUnixNano
	})

	return &tempopb.SearchResponse{
		Traces: results,
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: sr.TracesInspected(),
			InspectedBytes:  sr.BytesInspected(),
		},
	}, nil
}

// searchWAL starts a search task for every WAL block. Must be called under lock.
func (i *instance) searchWAL(ctx context.Context, req *tempopb.SearchRequest, sr *search.Results) {
	searchWalBlock := func(b common.WALBlock) {
		blockID := b.BlockMeta().BlockID.String()
		span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchWALBlock", opentracing.Tags{
			"blockID": blockID,
		})
		defer span.Finish()
		defer sr.FinishWorker()

		var resp *tempopb.SearchResponse
		var err error

		opts := common.DefaultSearchOptions()
		if api.IsTraceQLQuery(req) {
			// note: we are creating new engine for each wal block,
			// and engine.ExecuteSearch is parsing the query for each block
			resp, err = traceql.NewEngine().ExecuteSearch(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return b.Fetch(ctx, req, opts)
			}))
		} else {
			resp, err = b.Search(ctx, req, opts)
		}

		if err == common.ErrUnsupported {
			level.Warn(log.Logger).Log("msg", "wal block does not support search", "blockID", b.BlockMeta().BlockID)
			return
		}
		if err != nil {
			level.Error(log.Logger).Log("msg", "error searching local block", "blockID", blockID, "block_version", b.BlockMeta().Version, "err", err)
			sr.SetError(err)
			return
		}

		sr.AddBlockInspected()
		sr.AddBytesInspected(resp.Metrics.InspectedBytes)
		sr.AddTraceInspected(resp.Metrics.InspectedTraces)
		for _, r := range resp.Traces {
			sr.AddResult(ctx, r)
		}
	}

	// head block
	if i.headBlock != nil {
		sr.StartWorker()
		go searchWalBlock(i.headBlock)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		sr.StartWorker()
		go searchWalBlock(b)
	}
}

// searchLocalBlocks starts a search task for every local block. Must be called under lock.
func (i *instance) searchLocalBlocks(ctx context.Context, req *tempopb.SearchRequest, sr *search.Results) {
	// next check all complete blocks to see if they were not searched, if they weren't then attempt to search them
	for _, e := range i.completeBlocks {
		sr.StartWorker()
		go func(e *localBlock) {
			defer sr.FinishWorker()

			span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchLocalBlocks")
			defer span.Finish()

			blockID := e.BlockMeta().BlockID.String()

			span.LogFields(ot_log.Event("local block entry mtx acquired"))
			span.SetTag("blockID", blockID)

			var resp *tempopb.SearchResponse
			var err error

			opts := common.DefaultSearchOptions()
			if api.IsTraceQLQuery(req) {
				// note: we are creating new engine for each wal block,
				// and engine.ExecuteSearch is parsing the query for each block
				resp, err = traceql.NewEngine().ExecuteSearch(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
					return e.Fetch(ctx, req, opts)
				}))
			} else {
				resp, err = e.Search(ctx, req, opts)
			}

			if err == common.ErrUnsupported {
				level.Warn(log.Logger).Log("msg", "block does not support search", "blockID", e.BlockMeta().BlockID)
				return
			}
			if err != nil {
				level.Error(log.Logger).Log("msg", "error searching local block", "blockID", blockID, "err", err)
				sr.SetError(err)
				return
			}

			for _, t := range resp.Traces {
				sr.AddResult(ctx, t)
			}
			sr.AddBlockInspected()

			sr.AddBytesInspected(resp.Metrics.InspectedBytes)
			sr.AddTraceInspected(resp.Metrics.InspectedTraces)
		}(e)
	}
}

func (i *instance) SearchTags(ctx context.Context, scope string) (*tempopb.SearchTagsResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	// check if it's the special intrinsic scope
	if scope == api.ParamScopeIntrinsic {
		return &tempopb.SearchTagsResponse{
			TagNames: search.GetVirtualIntrinsicValues(),
		}, nil
	}

	// parse for normal scopes
	attributeScope := traceql.AttributeScopeFromString(scope)
	if attributeScope == traceql.AttributeScopeUnknown {
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	search := func(s common.Searcher, dv *util.DistinctStringCollector) error {
		if s == nil {
			return nil
		}
		if dv.Exceeded() {
			return nil
		}
		err = s.SearchTags(ctx, attributeScope, dv.Collect, common.DefaultSearchOptions())
		if err != nil && err != common.ErrUnsupported {
			return fmt.Errorf("unexpected error searching tags: %w", err)
		}

		return nil
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	// search parquet wal/completing blocks/completed blocks
	if err = search(i.headBlock, distinctValues); err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}
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
		level.Warn(log.Logger).Log("msg", "size of tags in instance exceeded limit, reduce cardinality or size of tags", "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	return &tempopb.SearchTagsResponse{
		TagNames: distinctValues.Strings(),
	}, nil
}

// SearchTagsV2 calls SearchTags for each scope and returns the results.
func (i *instance) SearchTagsV2(ctx context.Context, scope string) (*tempopb.SearchTagsV2Response, error) {
	scopes := []string{scope}
	if scope == "" {
		// start with intrinsic scope and all traceql attribute scopes
		atts := traceql.AllAttributeScopes()
		scopes = make([]string, 0, len(atts)+1) // +1 for intrinsic

		scopes = append(scopes, api.ParamScopeIntrinsic)
		for _, att := range atts {
			scopes = append(scopes, att.String())
		}
	}
	resps := make([]*tempopb.SearchTagsResponse, len(scopes))

	overallError := atomic.NewError(nil)
	wg := sync.WaitGroup{}
	for idx := range scopes {
		resps[idx] = &tempopb.SearchTagsResponse{}

		wg.Add(1)
		go func(scope string, ret **tempopb.SearchTagsResponse) {
			defer wg.Done()

			resp, err := i.SearchTags(ctx, scope)
			if err != nil {
				overallError.Store(fmt.Errorf("error searching tags: %s, %w", scope, err))
				return
			}

			*ret = resp
		}(scopes[idx], &resps[idx])
	}
	wg.Wait()

	err := overallError.Load()
	if err != nil {
		return nil, err
	}

	// build response
	resp := &tempopb.SearchTagsV2Response{}
	for idx := range resps {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scopes[idx],
			Tags: resps[idx].TagNames,
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
	distinctValues := util.NewDistinctStringCollector(limit)

	var inspectedBlocks, maxBlocks int
	if limit := i.limiter.limits.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = limit
	}

	search := func(s common.Searcher, dv *util.DistinctStringCollector) error {
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
		if err != nil && err != common.ErrUnsupported {
			return fmt.Errorf("unexpected error searching tag values (%s): %w", tagName, err)
		}

		return nil
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	// search parquet wal/completing blocks/completed blocks
	if err = search(i.headBlock, distinctValues); err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}
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

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctValueCollector[tempopb.TagValue](limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

	cb := func(v traceql.Static) bool {
		tv := tempopb.TagValue{}

		switch v.Type {
		case traceql.TypeString:
			tv.Type = "string"
			tv.Value = v.S // avoid formatting

		case traceql.TypeBoolean:
			tv.Type = "bool"
			tv.Value = v.String()

		case traceql.TypeInt:
			tv.Type = "int"
			tv.Value = v.String()

		case traceql.TypeFloat:
			tv.Type = "float"
			tv.Value = v.String()

		case traceql.TypeDuration:
			tv.Type = "duration"
			tv.Value = v.String()

		case traceql.TypeStatus:
			tv.Type = "keyword"
			tv.Value = v.String()
		}

		return distinctValues.Collect(tv)
	}

	engine := traceql.NewEngine()

	wg := boundedwaitgroup.New(20) // TODO: Make configurable
	var anyErr atomic.Error
	var inspectedBlocks atomic.Int32
	var maxBlocks int32
	if limit := i.limiter.limits.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = int32(limit)
	}

	searchBlock := func(s common.Searcher) error {
		if anyErr.Load() != nil {
			return nil // Early exit if any error has occurred
		}

		if maxBlocks > 0 && inspectedBlocks.Inc() > maxBlocks {
			return nil
		}

		fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return s.Fetch(ctx, req, common.DefaultSearchOptions())
		})
		return engine.ExecuteTagValues(ctx, req, cb, fetcher)
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	// completed blocks
	for _, b := range i.completeBlocks {
		wg.Add(1)
		go func(b *localBlock) {
			defer wg.Done()
			if err := searchBlock(b); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		wg.Add(1)
		go func(b common.WALBlock) {
			defer wg.Done()
			if err := searchBlock(b); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	// head block
	if i.headBlock != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := searchBlock(i.headBlock); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err))
			}
		}()
	}

	wg.Wait()

	if err := anyErr.Load(); err != nil {
		return nil, err
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
