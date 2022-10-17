package ingester

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/api"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/util"

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
		// TODO actually implement recent traceQL search
		return &tempopb.SearchResponse{
			Traces:  nil,
			Metrics: &tempopb.SearchMetrics{},
		}, nil
	}

	p := search.NewSearchPipeline(req)

	sr := search.NewResults()
	defer sr.Close()

	i.searchLiveTraces(ctx, p, sr)

	// Lock blocks mutex until all search tasks have been created. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	i.searchWAL(ctx, p, sr)
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
func (i *instance) searchWAL(ctx context.Context, p search.Pipeline, sr *search.Results) {
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

	// head block
	sr.StartWorker()
	go searchFunc(i.searchHeadBlock)

	// completing blocks
	for _, e := range i.searchAppendBlocks {
		sr.StartWorker()
		go searchFunc(e)
	}
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

			resp, err := e.Search(ctx, req, common.SearchOptions{})
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

			err = b.SearchTags(ctx, distinctValues.Collect, common.SearchOptions{})
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

			err = b.SearchTagValues(ctx, tagName, distinctValues.Collect, common.SearchOptions{})
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

	err := visitUnderLock(i.searchHeadBlock)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
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
