package combiner

import (
	"fmt"
	"io"
	"sort"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
)

var _ Combiner = (*genericCombiner[*tempopb.SearchResponse])(nil)

// TODO: we also have a combiner in pkg/traceql/combine.go, which is slightly different then this.
// this Combiner locks, and merges the spans slightly differently. compare and consolidate both if possible.
func NewSearch() Combiner {
	resultsMap := make(map[string]*tempopb.TraceSearchMetadata)
	return &genericCombiner[*tempopb.SearchResponse]{
		code:  200,
		final: &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(body io.ReadCloser, final *tempopb.SearchResponse) error {
			response := &tempopb.SearchResponse{}
			if err := jsonpb.Unmarshal(body, response); err != nil {
				return fmt.Errorf("error unmarshalling response body: %w", err)
			}
			for _, t := range response.Traces {
				if res := resultsMap[t.TraceID]; res != nil {
					// Merge search results
					CombineSearchResults(res, t)
				} else {
					// New entry
					resultsMap[t.TraceID] = t
				}
			}

			if response.Metrics != nil {
				final.Metrics.InspectedBytes += response.Metrics.InspectedBytes
				final.Metrics.InspectedTraces += response.Metrics.InspectedTraces
				final.Metrics.TotalBlocks += response.Metrics.TotalBlocks
				final.Metrics.CompletedJobs += response.Metrics.CompletedJobs
				final.Metrics.TotalJobs += response.Metrics.TotalJobs
				final.Metrics.TotalBlockBytes += response.Metrics.TotalBlockBytes
			}

			return nil
		},
		result: func(response *tempopb.SearchResponse) (string, error) {
			for _, t := range resultsMap {
				response.Traces = append(response.Traces, t)
			}
			sort.Slice(response.Traces, func(i, j int) bool {
				return response.Traces[i].StartTimeUnixNano > response.Traces[j].StartTimeUnixNano
			})

			return new(jsonpb.Marshaler).MarshalToString(response)
		},
	}
}

// TODO: merge this with /pkg/traceql/combine.go#L46-L95, this method is slightly different so look into it and merge both.
func CombineSearchResults(existing *tempopb.TraceSearchMetadata, incoming *tempopb.TraceSearchMetadata) {
	if existing.TraceID == "" {
		existing.TraceID = incoming.TraceID
	}

	if existing.RootServiceName == "" {
		existing.RootServiceName = incoming.RootServiceName
	}

	if existing.RootTraceName == "" {
		existing.RootTraceName = incoming.RootTraceName
	}

	// Earliest start time.
	if existing.StartTimeUnixNano > incoming.StartTimeUnixNano {
		existing.StartTimeUnixNano = incoming.StartTimeUnixNano
	}

	// Longest duration
	if existing.DurationMs < incoming.DurationMs {
		existing.DurationMs = incoming.DurationMs
	}

	// If TraceQL results are present
	if incoming.SpanSet != nil {
		if existing.SpanSet == nil {
			existing.SpanSet = &tempopb.SpanSet{}
		}

		existing.SpanSet.Matched += incoming.SpanSet.Matched
		existing.SpanSet.Spans = append(existing.SpanSet.Spans, incoming.SpanSet.Spans...)
		// Note - should we dedupe spans? Spans shouldn't be present in multiple clusters.
	}
}
