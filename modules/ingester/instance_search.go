package ingester

import (
	"context"
	"sort"

	"github.com/grafana/tempo/modules/search"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
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

	sr.SetWorkerCount(3)
	i.searchLiveTraces(ctx, p, sr)
	i.searchWAL(ctx, p, sr)
	i.searchLocalBlocks(ctx, p, sr)

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
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for _, s := range i.searchAppendBlocks {
		go func(s search.SearchBlock) {
			defer sr.FinishWorker()
			s.Search(ctx, p, sr)
		}(s)
	}
}

func (i *instance) searchLocalBlocks(ctx context.Context, p search.Pipeline, sr *search.SearchResults) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for _, s := range i.searchCompleteBlocks {
		go func(s search.SearchBlock) {
			defer sr.FinishWorker()
			s.Search(ctx, p, sr)
		}(s)
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
	tags := make([]string, 0, len(i.searchTagLookups))
	for k := range i.searchTagLookups {
		tags = append(tags, k)
	}
	return tags
}

func (i *instance) GetSearchTagValues(tagName string) []string {
	return i.searchTagLookups[tagName]
}

// Record first 50 unique values for every tag
const maxLookups = 50

func (i *instance) RecordSearchLookupValues(b []byte) {
	kv := &tempofb.KeyValues{}

	s := tempofb.SearchDataFromBytes(b)
	for j := 0; j < s.TagsLength(); j++ {
		s.Tags(kv, j)
		key := string(kv.Key())
		if vals, ok := i.searchTagLookups[key]; ok && len(vals) >= maxLookups {
			// key exists and we have enough
			continue
		}
		for k := 0; k < kv.ValueLength(); k++ {
			i.searchTagLookups.Add(key, string(kv.Value(k)))
		}
	}
}
