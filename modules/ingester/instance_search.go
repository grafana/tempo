package ingester

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/api"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/search"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/user"
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

	p := search.NewSearchPipeline(req)

	sr := search.NewResults()
	defer sr.Close() // signal all running workers to quit

	// skip live traces in TraceQL queries
	if !api.IsTraceQLQuery(req) {
		i.searchLiveTraces(ctx, p, sr)
	}

	// Lock blocks mutex until all search tasks have been created. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	i.searchWAL(ctx, req, p, sr)
	i.searchLocalBlocks(ctx, req, p, sr)
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
			InspectedBlocks: sr.BlocksInspected(),
			SkippedBlocks:   sr.BlocksSkipped(),
		},
	}, nil
}

func (i *instance) searchLiveTraces(ctx context.Context, p search.Pipeline, sr *search.Results) {
	sr.StartWorker()

	go func() {
		span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchLiveTraces")
		defer span.Finish()

		defer sr.FinishWorker()

		i.tracesMtx.Lock()
		defer i.tracesMtx.Unlock()

		span.LogFields(ot_log.Event("live traces mtx acquired"))

		entry := &tempofb.SearchEntry{} // buffer

		for _, t := range i.traces {
			if sr.Quit() {
				return
			}

			sr.AddTraceInspected(1)

			var result *tempopb.TraceSearchMetadata

			// Search and combine from all segments for the trace.
			for _, s := range t.searchData {
				sr.AddBytesInspected(uint64(len(s)))

				entry.Reset(s)
				if p.Matches(entry) {
					newResult := search.GetSearchResultFromData(entry)
					if result != nil {
						search.CombineSearchResults(result, newResult)
					} else {
						result = newResult
					}
				}
			}

			if result != nil {
				if quit := sr.AddResult(ctx, result); quit {
					return
				}
			}
		}
	}()
}

// searchWAL starts a search task for every WAL block. Must be called under lock.
func (i *instance) searchWAL(ctx context.Context, req *tempopb.SearchRequest, p search.Pipeline, sr *search.Results) {
	searchFBEntry := func(e *searchStreamingBlockEntry) {
		// flat-buffers search
		span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchWAL")
		defer span.Finish()

		defer sr.FinishWorker()

		e.mtx.RLock()
		defer e.mtx.RUnlock()

		span.LogFields(ot_log.Event("streaming block entry mtx acquired"))
		span.SetTag("blockID", e.b.BlockID().String())

		err := e.b.Search(ctx, p, sr)
		if err != nil {
			level.Error(log.Logger).Log("msg", "error searching wal block", "blockID", e.b.BlockID().String(), "err", err)
			sr.SetError(err)
		}
	}

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
			// and engine.Execute is parsing the query for each block
			resp, err = traceql.NewEngine().Execute(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return b.Fetch(ctx, req, opts)
			}))
		} else {
			resp, err = b.Search(ctx, req, opts)
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

	// skip flat-buffers search in TraceQL queries
	// sanity check, vParquet blocks shouldn't have flat-buffers search blocks
	if i.searchHeadBlock != nil && !api.IsTraceQLQuery(req) {
		sr.StartWorker()
		go searchFBEntry(i.searchHeadBlock)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		sr.StartWorker()
		go searchWalBlock(b)
	}

	// skip flat-buffers search in TraceQL queries
	// sanity check, vParquet blocks shouldn't have flat-buffers search blocks
	if !api.IsTraceQLQuery(req) {
		for _, e := range i.searchAppendBlocks {
			sr.StartWorker()
			go searchFBEntry(e)
		}
	}
}

// searchLocalBlocks starts a search task for every local block. Must be called under lock.
func (i *instance) searchLocalBlocks(ctx context.Context, req *tempopb.SearchRequest, p search.Pipeline, sr *search.Results) {
	if !api.IsTraceQLQuery(req) { // Skip flat-buffers search in TraceQL queries

		// first check the searchCompleteBlocks map. if there is an entry for a block here we want to search it first
		for _, e := range i.searchCompleteBlocks {
			sr.StartWorker()
			go func(e *searchLocalBlockEntry) {
				span, ctx := opentracing.StartSpanFromContext(ctx, "instance.fb.searchLocalBlocks")
				defer span.Finish()

				defer sr.FinishWorker()

				e.mtx.RLock()
				defer e.mtx.RUnlock()

				span.LogFields(ot_log.Event("local block entry mtx acquired"))
				span.SetTag("blockID", e.b.BlockID().String())

				// flat-buffers search
				err := e.b.Search(ctx, p, sr)
				if err != nil {
					level.Error(log.Logger).Log("msg", "error searching local block", "blockID", e.b.BlockID().String(), "err", err)
					sr.SetError(err)
				}
			}(e)
		}
	}

	// next check all complete blocks to see if they were not searched, if they weren't then attempt to search them
	for _, e := range i.completeBlocks {
		if _, ok := i.searchCompleteBlocks[e]; ok && !api.IsTraceQLQuery(req) {
			// no need to search this block, we already did above
			// only applies to non-traceql queries
			continue
		}

		// todo: remove support for v2 search and then this check can be removed.
		if e.BlockMeta().Version == v2.Encoding {
			level.Warn(log.Logger).Log("msg", "local block search not supported on v2 blocks")
			continue
		}

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
				// and engine.Execute is parsing the query for each block
				resp, err = traceql.NewEngine().Execute(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
					return e.Fetch(ctx, req, opts)
				}))
			} else {
				resp, err = e.Search(ctx, req, opts)
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

func (i *instance) SearchTags(ctx context.Context) (*tempopb.SearchTagsResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	// wal + search blocks
	if !distinctValues.Exceeded() {
		err = i.visitSearchableBlocks(ctx, func(block search.SearchableBlock) error {
			return block.Tags(ctx, distinctValues.Collect)
		})
		if err != nil {
			return nil, err
		}
	}

	search := func(s common.Searcher, dv *util.DistinctStringCollector) error {
		if s == nil {
			return nil
		}
		if dv.Exceeded() {
			return nil
		}
		err = s.SearchTags(ctx, dv.Collect, common.DefaultSearchOptions())
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
		_, ok := i.searchCompleteBlocks[b]
		if ok {
			// no need to search this block, we already did above
			continue
		}
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

func (i *instance) SearchTagValues(ctx context.Context, tagName string) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	// live traces
	kv := &tempofb.KeyValues{}
	tagNameBytes := []byte(tagName)
	err = i.visitSearchEntriesLiveTraces(ctx, func(entry *tempofb.SearchEntry) {
		kv := tempofb.FindTag(entry, kv, tagNameBytes)
		if kv != nil {
			for i, ii := 0, kv.ValueLength(); i < ii; i++ {
				key := string(kv.Value(i))
				distinctValues.Collect(key)
			}
		}
	})
	if err != nil {
		return nil, err
	}

	search := func(s common.Searcher, dv *util.DistinctStringCollector) error {
		if s == nil {
			return nil
		}
		if dv.Exceeded() {
			return nil
		}
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
		_, ok := i.searchCompleteBlocks[b]
		if ok {
			// no need to search this block, we already did above
			continue
		}
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

	tag, err := traceql.ParseIdentifier(req.TagName)
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

	search := func(s common.Searcher, dv *util.DistinctValueCollector[tempopb.TagValue]) error {
		if s == nil || dv.Exceeded() {
			return nil
		}

		err = s.SearchTagValuesV2(ctx, tag, cb, common.DefaultSearchOptions())
		if err != nil && err != common.ErrUnsupported {
			return fmt.Errorf("unexpected error searching tag values v2 (%s): %w", tag, err)
		}
		return nil
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()
	// head block
	if err = search(i.headBlock, distinctValues); err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}

	// completed blocks
	for _, b := range i.completeBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err)
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

func (i *instance) visitSearchEntriesLiveTraces(ctx context.Context, visitFn func(entry *tempofb.SearchEntry)) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "instance.visitSearchEntriesLiveTraces")
	defer span.Finish()

	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	se := &tempofb.SearchEntry{}
	for _, t := range i.traces {
		for _, s := range t.searchData {
			se.Reset(s)
			visitFn(se)

			if err := ctx.Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *instance) visitSearchableBlocks(ctx context.Context, visitFn func(block search.SearchableBlock) error) error {
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	err := i.visitSearchableBlocksWAL(ctx, visitFn)
	if err != nil {
		return err
	}

	return i.visitSearchableBlocksLocalBlocks(ctx, visitFn)
}

// visitSearchableBlocksWAL visits every WAL block. Must be called under lock.
func (i *instance) visitSearchableBlocksWAL(ctx context.Context, visitFn func(block search.SearchableBlock) error) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "instance.visitSearchableBlocksWAL")
	defer span.Finish()

	visitUnderLock := func(entry *searchStreamingBlockEntry) error {
		entry.mtx.RLock()
		defer entry.mtx.RUnlock()

		return visitFn(entry.b)
	}

	if i.searchHeadBlock != nil {
		err := visitUnderLock(i.searchHeadBlock)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	for _, b := range i.searchAppendBlocks {
		err := visitUnderLock(b)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

// visitSearchableBlocksWAL visits every local block. Must be called under lock.
func (i *instance) visitSearchableBlocksLocalBlocks(ctx context.Context, visitFn func(block search.SearchableBlock) error) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "instance.visitSearchableBlocksLocalBlocks")
	defer span.Finish()

	visitUnderLock := func(entry *searchLocalBlockEntry) error {
		entry.mtx.RLock()
		defer entry.mtx.RUnlock()

		return visitFn(entry.b)
	}

	for _, b := range i.searchCompleteBlocks {
		err := visitUnderLock(b)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}
