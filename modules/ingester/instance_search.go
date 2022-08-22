package ingester

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-kit/log/level"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/util"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/search"
	"github.com/grafana/tempo/tempodb/wal"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	maxResults := int(req.Limit)
	// if limit is not set, use a safe default
	if maxResults == 0 {
		maxResults = 20
	}

	p := search.NewSearchPipeline(req)

	sr := search.NewResults()
	defer sr.Close()

	i.searchLiveTraces(ctx, p, sr)

	// Lock blocks mutex until all search tasks have been created. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()
	i.searchWAL(ctx, p, sr)
	i.searchLocalBlocks(ctx, req, sr)

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
func (i *instance) searchLocalBlocks(ctx context.Context, req *tempopb.SearchRequest, sr *search.Results) {
	for _, e := range i.completeBlocks {
		if e.BlockMeta().Version == v2.Encoding {
			level.Warn(log.Logger).Log("msg", "local block search not supported on v2 blocks")
			continue
		}

		sr.StartWorker()
		go func(e *wal.LocalBlock) {
			defer sr.FinishWorker()

			span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchLocalBlocks")
			defer span.Finish()

			blockID := e.BlockMeta().BlockID

			span.LogFields(ot_log.Event("local block entry mtx acquired"))
			span.SetTag("blockID", blockID)

			resp, err := e.Search(ctx, req, common.SearchOptions{}) // jpe options? buffers = 0 for no buffered reader at
			if err != nil {
				level.Error(log.Logger).Log("msg", "error searching local block", "blockID", blockID, "err", err)
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

	// wal
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
			err = b.SearchTags(ctx, distinctValues.Collect, common.SearchOptions{}) // jpe !!
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

// jpe add search tag values functionality? - skip unsupported blocks
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

	// wal
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
			err = b.SearchTagValues(ctx, tagName, distinctValues.Collect, common.SearchOptions{}) // jpe !!
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

	return i.visitSearchableBlocksWAL(ctx, visitFn)
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
