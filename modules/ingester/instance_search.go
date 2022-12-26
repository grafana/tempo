package ingester

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/api"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/search"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	maxResults := int(req.Limit)
	// if limit is not set, use a safe default
	if maxResults == 0 {
		maxResults = 20
	}

	if api.IsTraceQLQuery(req) {
		// TODO: traceQL only works with vParquet WAL??
		// should we check and return an error if we are not using vParquet WAL??
		return i.searchWALWithTraceQL(ctx, req, maxResults)
	}

	p := search.NewSearchPipeline(req)

	sr := search.NewResults()
	defer sr.Close()

	i.searchLiveTraces(ctx, p, sr)

	// Lock blocks mutex until all search tasks have been created. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	i.searchWAL(ctx, req, p, sr)
	i.searchLocalBlocks(ctx, req, p, sr)
	i.blocksMtx.RUnlock()

	sr.AllWorkersStarted()

	resultsMap := map[string]*tempopb.TraceSearchMetadata{}

	for result := range sr.Results() {
		// Dedupe/combine results
		if existing := resultsMap[result.TraceID]; existing != nil {
			search.CombineSearchResults(existing, result)
		} else {
			resultsMap[result.TraceID] = result
		}

		if len(resultsMap) >= maxResults {
			break
		}
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
	searchFunc := func(e *searchStreamingBlockEntry) {
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
		}
	}

	searchWalBlock := func(b common.WALBlock) {
		span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchWALBlock", opentracing.Tags{
			"blockID": b.BlockMeta().BlockID,
		})
		defer span.Finish()
		defer sr.FinishWorker()

		resp, err := b.Search(ctx, req, common.DefaultSearchOptions())
		if err != nil {
			level.Error(log.Logger).Log("msg", "error searching wal block", "blockID", b.BlockMeta().BlockID.String(), "err", err)
			return
		}

		sr.AddBlockInspected()
		sr.AddBytesInspected(resp.Metrics.InspectedBytes)
		for _, r := range resp.Traces {
			sr.AddResult(ctx, r)
		}
	}

	// head block
	if i.headBlock != nil {
		sr.StartWorker()
		go searchWalBlock(i.headBlock)
	}

	if i.searchHeadBlock != nil {
		sr.StartWorker()
		go searchFunc(i.searchHeadBlock)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		sr.StartWorker()
		go searchWalBlock(b)
	}

	for _, e := range i.searchAppendBlocks {
		sr.StartWorker()
		go searchFunc(e)
	}
}

// searchWALWithTraceQL handles TraceQL search query for searching WAL,
// this func takes blocksMtx when searching WAL blocks
// FIXME: rename func to executeTraceQLonWAL, and take a parsed query
func (i *instance) searchWALWithTraceQL(ctx context.Context, req *tempopb.SearchRequest, maxResults int) (*tempopb.SearchResponse, error) {
	// A trace can be a live, in headBlock, completing blocks and complete blocks.
	// search in this order: HeadBlock, CompletingBlocks, and then finally in completeBlocks.
	// Iterate over blocks, execute TraceQL, collect results, dedupe, sort and return traces.
	// exit if we hit maxResults during search.

	// TODO: instrument this method with tracing
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchWALWithTraceQL")
	defer span.Finish()

	// fmt.Printf("------------------------------\n")
	// fmt.Printf("TraceQLQuery: %v\n", req)

	opts := common.DefaultSearchOptions()
	engine := traceql.NewEngine()

	// collect traces and metrics
	var traces []*tempopb.TraceSearchMetadata
	var metrics []*tempopb.SearchMetrics

	// i.traces has live Traces
	// TODO: searching live traces with TraceQL is not supported yet

	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	span.LogFields(ot_log.Event("blocksMtx acquired"))

	// span.LogFields(
	// 	ot_log.Int("failedBlocks", failedBlocks),
	// 	ot_log.Error(multierr.Combine(blockErrs...)),
	// )
	// _ = level.Warn(log.Logger).Log("msg", fmt.Sprintf("failed to query %d blocks", failedBlocks), "blockErrs", multierr.Combine(blockErrs...))

	// search headBlock
	// fmt.Printf("headBlock: %v\n", i.headBlock)

	// fmt.Printf("searching headBlock\n")
	res, err := engine.Execute(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		// span.SetTag("blockID", i.headBlock.BlockMeta().BlockID.String())
		return i.headBlock.Fetch(ctx, req, opts)
	}))
	if err != nil {
		// TODO: log error and return??
		return nil, err
	}

	// collect headBlock results
	traces = append(traces, res.Traces...)
	metrics = append(metrics, res.Metrics)

	// exit early
	if len(traces) >= maxResults {
		// fmt.Printf("getting out early after headBlock\n")
		// TODO: return metrics and error??
		return res, nil
	}

	// fmt.Printf("res: %v, err: %v\n", res, err)

	// fmt.Printf("completingBlocks: %v\n", i.completingBlocks)
	// search completingBlocks
	for _, block := range i.completingBlocks {
		// fmt.Printf("searching completingBlocks\n")
		r, err := engine.Execute(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return block.Fetch(ctx, req, opts)
		}))
		if err != nil {
			// fmt.Printf("error in searching completingBlocks: %v\n", err)
			span.LogFields(ot_log.Error(fmt.Errorf("error searching completingBlocks: blockID %v: %w", block.BlockMeta().BlockID.String(), err)))
			// TODO: should we fail the search if we fail to query single block in WAL??
			continue
		}

		// fmt.Printf("traces: %v\n", r.Traces)
		// fmt.Printf("metrics: %v\n", r.Metrics)

		traces = append(traces, r.Traces...)
		metrics = append(metrics, r.Metrics)

		// exit early
		if len(traces) >= maxResults {
			// fmt.Printf("getting out early from completingBlocks\n")
			break
		}
	}

	// search completeBlocks
	// fmt.Printf("completeBlocks: %v\n", i.completeBlocks)
	for _, block := range i.completeBlocks {
		// fmt.Printf("searching completeBlocks\n")
		// TODO: Execute will parse the query again and again... maybe fix that
		// TODO: we are building NewSpansetFetcherWrapper for each block, feels bit wasteful :)
		r, err := engine.Execute(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return block.Fetch(ctx, req, opts)
		}))

		if err != nil {
			// fmt.Printf("error in searching completeBlocks: %v\n", err)
			span.LogFields(ot_log.Error(fmt.Errorf("error searching completeBlocks: blockID %v: %w", block.BlockMeta().BlockID.String(), err)))
			// TODO: should we fail the search if we fail to query single block in WAL??
			continue
		}

		// fmt.Printf("traces len: %d \n", len(traces))
		// fmt.Printf("metrics: %v\n", r.Metrics)

		traces = append(traces, r.Traces...)
		metrics = append(metrics, r.Metrics)

		// exit early from search
		if len(traces) >= maxResults {
			// fmt.Printf("getting out early from completeBlocks\n")
			break
		}
	}

	// fmt.Printf("len of found traces: %v \n", len(traces))

	// if len(traces) == 20 {
	// 	for _, trace := range traces {
	// 		fmt.Printf("traceID: %s\n", trace.TraceID)
	// 	}
	// }

	// FIXME: move this out of this func
	// de-duplicate and sort results
	// TODO: this du-dupe and sort code is copy of v2 search results de-dupe code

	// FIXME: Write a tempopb.SearchResponse combiner ???
	// take results from calling engine.Execute and builds a de-duped results
	// create a new combiner, and use it here and any other place where we need to combine SearchResponse
	// when combining SearchResponse, we need to merge traces and metrics both.
	// combiner merges traces and metrics on the fly so we can exit early???
	resultsMap := map[string]*tempopb.TraceSearchMetadata{}
	for _, result := range traces {
		// Dedupe/combine results
		if existing := resultsMap[result.TraceID]; existing != nil {
			search.CombineSearchResults(existing, result)
		} else {
			resultsMap[result.TraceID] = result
		}

		if len(resultsMap) >= maxResults {
			// fmt.Printf("hit maxResults during de-duplication\n")
			break
		}
	}

	results := make([]*tempopb.TraceSearchMetadata, 0, len(resultsMap))
	for _, result := range resultsMap {
		results = append(results, result)
	}

	// Sort
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartTimeUnixNano > results[j].StartTimeUnixNano
	})

	// TODO: merge search result metrics
	// finalMetrics := &tempopb.SearchMetrics{}
	// for _, m := range metrics {
	// 	finalMetrics.InspectedTraces += m.InspectedTraces
	// 	finalMetrics.InspectedBytes += m.InspectedBytes
	// 	finalMetrics.InspectedBlocks += m.InspectedBlocks
	// 	finalMetrics.SkippedBlocks += m.SkippedBlocks
	// 	finalMetrics.SkippedTraces += m.SkippedTraces
	// 	finalMetrics.TotalBlockBytes += m.TotalBlockBytes
	// }

	return &tempopb.SearchResponse{
		Traces:  results,
		Metrics: metrics[0],
	}, nil
}

// searchLocalBlocks starts a search task for every local block. Must be called under lock.
func (i *instance) searchLocalBlocks(ctx context.Context, req *tempopb.SearchRequest, p search.Pipeline, sr *search.Results) {
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

			err := e.b.Search(ctx, p, sr)
			if err != nil {
				level.Error(log.Logger).Log("msg", "error searching local block", "blockID", e.b.BlockID().String(), "err", err)
			}
		}(e)
	}

	// next check all complete blocks to see if they were not searched, if they weren't then attempt to search them
	for _, e := range i.completeBlocks {
		_, ok := i.searchCompleteBlocks[e]
		if ok {
			// no need to search this block, we already did above
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

			blockID := e.BlockMeta().BlockID

			span.LogFields(ot_log.Event("local block entry mtx acquired"))
			span.SetTag("blockID", blockID)

			resp, err := e.Search(ctx, req, common.DefaultSearchOptions())
			if err != nil {
				level.Error(log.Logger).Log("msg", "error searching local block", "blockID", blockID, "err", err)
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

	// live traces
	kv := &tempofb.KeyValues{}
	err = i.visitSearchEntriesLiveTraces(ctx, func(entry *tempofb.SearchEntry) {
		for i, ii := 0, entry.TagsLength(); i < ii; i++ {
			entry.Tags(kv, i)
			key := string(kv.Key())
			distinctValues.Collect(key)
		}
	})
	if err != nil {
		return nil, err
	}

	// wal + search blocks
	if !distinctValues.Exceeded() {
		err = i.visitSearchableBlocks(ctx, func(block search.SearchableBlock) error {
			return block.Tags(ctx, distinctValues.Collect)
		})
		if err != nil {
			return nil, err
		}
	}

	// local blocks
	if !distinctValues.Exceeded() {
		i.blocksMtx.RLock()
		defer i.blocksMtx.RUnlock()
		for _, b := range i.completeBlocks {
			_, ok := i.searchCompleteBlocks[b]
			if ok {
				// no need to search this block, we already did above
				continue
			}

			err = b.SearchTags(ctx, distinctValues.Collect, common.DefaultSearchOptions())
			if err == common.ErrUnsupported {
				level.Warn(log.Logger).Log("msg", "block does not support tag search", "blockID", b.BlockMeta().BlockID)
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("unexpected error searching tags (%s): %w", b.BlockMeta().BlockID, err)
			}
			if distinctValues.Exceeded() {
				break
			}
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

	// wal + search blocks
	if !distinctValues.Exceeded() {
		err = i.visitSearchableBlocks(ctx, func(block search.SearchableBlock) error {
			return block.TagValues(ctx, tagName, distinctValues.Collect)
		})
		if err != nil {
			return nil, err
		}
	}

	// local blocks
	if !distinctValues.Exceeded() {
		i.blocksMtx.RLock()
		defer i.blocksMtx.RUnlock()
		for _, b := range i.completeBlocks {
			_, ok := i.searchCompleteBlocks[b]
			if ok {
				// no need to search this block, we already did above
				continue
			}

			err = b.SearchTagValues(ctx, tagName, distinctValues.Collect, common.DefaultSearchOptions())
			if err == common.ErrUnsupported {
				level.Warn(log.Logger).Log("msg", "block does not support tag value search", "blockID", b.BlockMeta().BlockID)
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("unexpected error searching tag values (%s): %w", b.BlockMeta().BlockID, err)
			}
			if distinctValues.Exceeded() {
				break
			}
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", tagName, "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	return &tempopb.SearchTagValuesResponse{
		TagValues: distinctValues.Strings(),
	}, nil
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
