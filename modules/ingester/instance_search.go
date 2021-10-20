package ingester

import (
	"context"
	"fmt"
	"sort"
	"time"

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
		span.SetTag("blockID", e.b.BlockID.String())

		err := e.b.Search(ctx, p, sr)
		if err != nil {
			fmt.Println("error searching wal block", e.b.BlockID.String(), err)
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
				fmt.Println("error searching local block", e.b.BlockID().String(), err)
			}
		}(e)
	}
}

func (i *instance) GetSearchTags() []string {
	return i.searchTagCache.GetNames()
}

func (i *instance) GetSearchTagValues(tagName string) []string {
	return i.searchTagCache.GetValues(tagName)
}

func (i *instance) RecordSearchLookupValues(b []byte) {
	s := tempofb.SearchEntryFromBytes(b)
	i.searchTagCache.SetData(time.Now(), s)
}

func (i *instance) PurgeExpiredSearchTags(before time.Time) {
	i.searchTagCache.PurgeExpired(before)
}
