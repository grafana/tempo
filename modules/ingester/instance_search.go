package ingester

import (
	"context"
	"sort"

	cortex_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
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

	p := search.NewSearchPipeline(req)

	sr := search.NewResults()
	defer sr.Close()

	i.searchLiveTraces(ctx, p, sr)

	// Lock blocks mutex until all search tasks have been created. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	i.searchWAL(ctx, p, sr)
	i.searchLocalBlocks(ctx, p, sr)
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

		for _, t := range i.traces {
			if sr.Quit() {
				return
			}

			sr.AddTraceInspected(1)

			var result *tempopb.TraceSearchMetadata

			// Search and combine from all segments for the trace.
			for _, s := range t.searchData {
				sr.AddBytesInspected(uint64(len(s)))

				entry := tempofb.SearchEntryFromBytes(s)
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
			level.Error(cortex_util.Logger).Log("msg", "error searching wal block", "blockID", e.b.BlockID().String(), "err", err)
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
func (i *instance) searchLocalBlocks(ctx context.Context, p search.Pipeline, sr *search.Results) {
	for _, e := range i.searchCompleteBlocks {
		sr.StartWorker()
		go func(e *searchLocalBlockEntry) {
			span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchLocalBlocks")
			defer span.Finish()

			defer sr.FinishWorker()

			e.mtx.RLock()
			defer e.mtx.RUnlock()

			span.LogFields(ot_log.Event("local block entry mtx acquired"))
			span.SetTag("blockID", e.b.BlockID().String())

			err := e.b.Search(ctx, p, sr)
			if err != nil {
				level.Error(cortex_util.Logger).Log("msg", "error searching local block", "blockID", e.b.BlockID().String(), "err", err)
			}
		}(e)
	}
}

func (i *instance) GetSearchTags(ctx context.Context) ([]string, error) {
	tags := map[string]struct{}{}

	i.visitSearchEntriesLiveTraces(ctx, func(entry *tempofb.SearchEntry) {
		kv := &tempofb.KeyValues{}

		for i, ii := 0, entry.TagsLength(); i < ii; i++ {
			entry.Tags(kv, i)
			tags[string(kv.Key())] = struct{}{}
		}
	})

	extractTagsFromSearchableBlock := func(block search.SearchableBlock) error {
		return block.Tags(ctx, tags)
	}
	err := func() error {
		i.blocksMtx.RLock()
		defer i.blocksMtx.RUnlock()

		err := i.visitSearchableBlocksWAL(ctx, extractTagsFromSearchableBlock)
		if err != nil {
			return err
		}
		return i.visitSearchableBlocksLocalBlocks(ctx, extractTagsFromSearchableBlock)
	}()
	if err != nil {
		return nil, err
	}

	tagsSlice := make([]string, 0, len(tags))
	for tag := range tags {
		tagsSlice = append(tagsSlice, tag)
	}

	return tagsSlice, nil
}

func (i *instance) GetSearchTagValues(ctx context.Context, tagName string) ([]string, error) {
	values := map[string]struct{}{}

	i.visitSearchEntriesLiveTraces(ctx, func(entry *tempofb.SearchEntry) {
		kv := &tempofb.KeyValues{}

		for i, tagsLength := 0, entry.TagsLength(); i < tagsLength; i++ {
			entry.Tags(kv, i)

			if string(kv.Key()) == tagName {
				for j, valueLength := 0, kv.ValueLength(); j < valueLength; j++ {
					values[string(kv.Value(j))] = struct{}{}
				}
				break
			}
		}
	})

	extractTagValuesFromSearchableBlocks := func(block search.SearchableBlock) error {
		return block.TagValues(ctx, tagName, values)
	}

	err := func() error {
		i.blocksMtx.RLock()
		defer i.blocksMtx.RUnlock()

		err := i.visitSearchableBlocksWAL(ctx, extractTagValuesFromSearchableBlocks)
		if err != nil {
			return err
		}
		return i.visitSearchableBlocksLocalBlocks(ctx, extractTagValuesFromSearchableBlocks)
	}()
	if err != nil {
		return nil, err
	}

	valuesSlice := make([]string, 0, len(values))
	for tag := range values {
		valuesSlice = append(valuesSlice, tag)
	}

	return valuesSlice, nil
}

func (i *instance) visitSearchEntriesLiveTraces(ctx context.Context, visit func(entry *tempofb.SearchEntry)) {
	span, _ := opentracing.StartSpanFromContext(ctx, "instance.visitSearchEntriesLiveTraces")
	defer span.Finish()

	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	for _, t := range i.traces {
		for _, s := range t.searchData {
			visit(tempofb.SearchEntryFromBytes(s))
		}
	}
}

// visitSearchableBlocksWAL visits every WAL block. Must be called under lock.
func (i *instance) visitSearchableBlocksWAL(ctx context.Context, visit func(block search.SearchableBlock) error) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "instance.visitSearchableBlocksWAL")
	defer span.Finish()

	visitUnderLock := func(entry *searchStreamingBlockEntry) error {
		entry.mtx.RLock()
		defer entry.mtx.RUnlock()

		return visit(entry.b)
	}

	err := visitUnderLock(i.searchHeadBlock)
	if err != nil {
		return err
	}

	for _, b := range i.searchAppendBlocks {
		err := visitUnderLock(b)
		if err != nil {
			return err
		}
	}
	return nil
}

// visitSearchableBlocksWAL visits every local block. Must be called under lock.
func (i *instance) visitSearchableBlocksLocalBlocks(ctx context.Context, visit func(block search.SearchableBlock) error) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "instance.visitSearchableBlocksLocalBlocks")
	defer span.Finish()

	visitUnderLock := func(entry *searchLocalBlockEntry) error {
		entry.mtx.RLock()
		defer entry.mtx.RUnlock()

		return visit(entry.b)
	}

	for _, b := range i.searchCompleteBlocks {
		err := visitUnderLock(b)
		if err != nil {
			return err
		}
	}
	return nil
}
