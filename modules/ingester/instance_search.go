package ingester

import (
	"context"
	"sort"
	"sync"

	"github.com/grafana/tempo/modules/search"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) ([]*tempopb.TraceSearchMetadata, error) {

	done := make(chan struct{})
	defer close(done)

	var resultsCh []<-chan *tempopb.TraceSearchMetadata

	// TODO - Redo this entire thing around channels and concurrent searches, and bail after reading
	// max results from channel
	maxResults := 20
	if req.Limit != 0 {
		maxResults = int(req.Limit)
	}

	var results []*tempopb.TraceSearchMetadata

	p := search.NewSearchPipeline(req)

	resultsCh = append(resultsCh, i.searchLiveTraces(ctx, done, p))
	resultsCh = append(resultsCh, i.searchWAL(ctx, done, p))
	resultsCh = append(resultsCh, i.searchLocalBlocks(ctx, done, p)...)

	for result := range mergeSearchResults(done, resultsCh...) {
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

	return results, nil
}

func (i *instance) searchLiveTraces(ctx context.Context, done <-chan struct{}, p search.Pipeline) <-chan *tempopb.TraceSearchMetadata {
	out := make(chan *tempopb.TraceSearchMetadata)

	go func() {
		defer close(out)

		i.tracesMtx.Lock()
		defer i.tracesMtx.Unlock()

		for _, t := range i.traces {
			for _, s := range t.searchData {
				searchData := tempofb.SearchDataFromBytes(s)
				if p.Matches(searchData) {
					result := search.GetSearchResultFromData(searchData)

					select {
					case out <- result:
					case <-ctx.Done():
						return
					case <-done:
						return
					}

					continue
				}
			}
		}
	}()

	return out
}

func (i *instance) searchWAL(ctx context.Context, done <-chan struct{}, p search.Pipeline) <-chan *tempopb.TraceSearchMetadata {
	out := make(chan *tempopb.TraceSearchMetadata)

	go func() {
		defer close(out)

		i.blocksMtx.Lock()
		defer i.blocksMtx.Unlock()

		for _, s := range i.searchAppendBlocks {
			headResults, err := s.Search(ctx, p)
			if err != nil {
				continue
			}

			for _, result := range headResults {
				select {
				case out <- result:
				case <-ctx.Done():
					return
				case <-done:
					return
				}
			}
		}
	}()

	return out
}

func (i *instance) searchLocalBlocks(ctx context.Context, done <-chan struct{}, p search.Pipeline) []<-chan *tempopb.TraceSearchMetadata {
	var outs []<-chan *tempopb.TraceSearchMetadata

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	for _, s := range i.searchCompleteBlocks {
		out := make(chan *tempopb.TraceSearchMetadata)
		outs = append(outs, out)

		go func(c chan *tempopb.TraceSearchMetadata, s search.SearchBlock) {
			defer close(c)

			results, err := s.Search(ctx, p)
			if err != nil {
				return
			}

			for _, result := range results {
				select {
				case out <- result:
				case <-ctx.Done():
					return
				case <-done:
					return
				}
			}
		}(out, s)
	}

	return outs
}

func mergeSearchResults(done <-chan struct{}, cs ...<-chan *tempopb.TraceSearchMetadata) <-chan *tempopb.TraceSearchMetadata {
	var wg sync.WaitGroup
	out := make(chan *tempopb.TraceSearchMetadata)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan *tempopb.TraceSearchMetadata) {
		for n := range c {
			select {
			case out <- n:
			case <-done:
			}
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
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
