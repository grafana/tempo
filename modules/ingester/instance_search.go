package ingester

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/search"
	"github.com/grafana/tempo/tempodb/wal"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {

	maxResults := int(req.Limit)

	p := search.NewSearchPipeline(req)

	sr := search.NewResults()
	defer sr.Close()

	i.searchLiveTraces(ctx, p, sr)
	i.searchWAL(ctx, p, sr)
	i.searchLocalBlocks(ctx, p, sr)

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

func (i *instance) searchWAL(ctx context.Context, p search.Pipeline, sr *search.Results) {
	searchFunc := func(k *wal.AppendBlock, e *searchStreamingBlockEntry) {
		defer sr.FinishWorker()

		e.mtx.RLock()
		defer e.mtx.RUnlock()

		err := e.b.Search(ctx, p, sr)
		if err != nil {
			fmt.Println("error searching wal block", k.BlockID().String(), err)
		}
	}

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	// head block
	sr.StartWorker()
	go searchFunc(i.headBlock, i.searchHeadBlock)

	// completing blocks
	for b, e := range i.searchAppendBlocks {
		sr.StartWorker()
		go searchFunc(b, e)
	}
}

func (i *instance) searchLocalBlocks(ctx context.Context, p search.Pipeline, sr *search.Results) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for b, e := range i.searchCompleteBlocks {
		sr.StartWorker()
		go func(b *wal.LocalBlock, e *searchLocalBlockEntry) {
			defer sr.FinishWorker()

			e.mtx.RLock()
			defer e.mtx.RUnlock()

			err := e.b.Search(ctx, p, sr)
			if err != nil {
				fmt.Println("error searching local block", b.BlockMeta().BlockID.String(), err)
			}
		}(b, e)
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
