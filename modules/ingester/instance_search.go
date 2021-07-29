package ingester

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/grafana/tempo/modules/search"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/wal"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {

	maxResults := 20
	if req.Limit != 0 {
		maxResults = int(req.Limit)
	}

	var results []*tempopb.TraceSearchMetadata

	p := search.NewSearchPipeline(req)

	sr := search.NewSearchResults()
	defer sr.Close()

	i.searchLiveTraces(ctx, p, sr)
	i.searchWAL(ctx, p, sr)
	i.searchLocalBlocks(ctx, p, sr)

	sr.AllWorkersStarted()

	for result := range sr.Results() {
		results = append(results, result)

		// Sort, dedupe and limit results
		sort.Slice(results, func(i, j int) bool {
			return results[i].StartTimeUnixNano > results[j].StartTimeUnixNano
		})

		results = dedupeResults(results)

		if len(results) >= maxResults {
			break
		}
	}

	return &tempopb.SearchResponse{
		Traces: results,
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: sr.TracesInspected(),
			InspectedBytes:  sr.BytesInspected(),
		},
	}, nil
}

func (i *instance) searchLiveTraces(ctx context.Context, p search.Pipeline, sr *search.SearchResults) {
	sr.StartWorker()

	go func() {
		defer sr.FinishWorker()

		i.tracesMtx.Lock()
		defer i.tracesMtx.Unlock()

		for _, t := range i.traces {
			if sr.Quit() {
				return
			}

			sr.AddTraceInspected()

			for _, s := range t.searchData {
				sr.AddBytesInspected(uint64(len(s)))

				searchData := tempofb.SearchDataFromBytes(s)
				if p.Matches(searchData) {
					result := search.GetSearchResultFromData(searchData)

					if quit := sr.AddResult(ctx, result); quit {
						return
					}

					continue
				}
			}
		}
	}()
}

func (i *instance) searchWAL(ctx context.Context, p search.Pipeline, sr *search.SearchResults) {
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

func (i *instance) searchLocalBlocks(ctx context.Context, p search.Pipeline, sr *search.SearchResults) {
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

func dedupeResults(results []*tempopb.TraceSearchMetadata) []*tempopb.TraceSearchMetadata {
	for i := range results {
		for j := i + 1; j < len(results); j++ {
			if results[i].TraceID == results[j].TraceID {
				results = append(results[:j], results[j+1:]...)
				j--
			}
		}
	}
	return results
}

func (i *instance) GetSearchTags() []string {
	return i.searchTagCache.GetNames()
}

func (i *instance) GetSearchTagValues(tagName string) []string {
	return i.searchTagCache.GetValues(tagName)
}

func (i *instance) RecordSearchLookupValues(b []byte) {
	s := tempofb.SearchDataFromBytes(b)
	i.searchTagCache.SetData(time.Now(), s)
}

func (i *instance) PurgeExpiredSearchTags(before time.Time) {
	i.searchTagCache.PurgeExpired(before)
}
