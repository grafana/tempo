package combiner

import (
	"sort"

	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

var _ GRPCCombiner[*tempopb.SearchResponse] = (*genericCombiner[*tempopb.SearchResponse])(nil)

// NewSearch returns a search combiner
func NewSearch(limit int) Combiner {
	metadataCombiner := traceql.NewMetadataCombiner()
	diffTraces := map[string]struct{}{}

	return &genericCombiner[*tempopb.SearchResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchResponse { return &tempopb.SearchResponse{} },
		current:        &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.SearchResponse, final *tempopb.SearchResponse) error {
			for _, t := range partial.Traces {
				// if we've reached the limit and this is NOT a new trace then skip it
				if limit > 0 &&
					metadataCombiner.Count() >= limit &&
					!metadataCombiner.Exists(t.TraceID) {
					continue
				}

				metadataCombiner.AddMetadata(t)
				// record modified traces
				diffTraces[t.TraceID] = struct{}{}
			}

			if partial.Metrics != nil {
				// there is a coordination with the search sharder here. normal responses
				// will never have total jobs set, but they will have valid Inspected* values
				// a special response is sent back from the sharder with no traces but valid Total* values
				// if TotalJobs is nonzero then assume its the special response
				if partial.Metrics.TotalJobs == 0 {
					final.Metrics.CompletedJobs++

					final.Metrics.InspectedBytes += partial.Metrics.InspectedBytes
					final.Metrics.InspectedTraces += partial.Metrics.InspectedTraces
				} else {
					final.Metrics.TotalBlocks += partial.Metrics.TotalBlocks
					final.Metrics.TotalJobs += partial.Metrics.TotalJobs
					final.Metrics.TotalBlockBytes += partial.Metrics.TotalBlockBytes
				}
			}

			return nil
		},
		finalize: func(final *tempopb.SearchResponse) (*tempopb.SearchResponse, error) {
			// metrics are already combined on the passed in final
			final.Traces = metadataCombiner.Metadata()

			addRootSpanNotReceivedText(final.Traces)
			return final, nil
		},
		diff: func(current *tempopb.SearchResponse) (*tempopb.SearchResponse, error) {
			// wipe out any existing traces and recreate from the map
			diff := &tempopb.SearchResponse{
				Traces:  make([]*tempopb.TraceSearchMetadata, 0, len(diffTraces)),
				Metrics: current.Metrics,
			}

			for _, tr := range metadataCombiner.Metadata() {
				// if not in the map, skip. we haven't seen an update
				if _, ok := diffTraces[tr.TraceID]; !ok {
					continue
				}

				diff.Traces = append(diff.Traces, tr)
			}

			sort.Slice(diff.Traces, func(i, j int) bool {
				return diff.Traces[i].StartTimeUnixNano > diff.Traces[j].StartTimeUnixNano
			})

			addRootSpanNotReceivedText(diff.Traces)

			// wipe out diff traces for the next time
			clear(diffTraces)

			return diff, nil
		},
		// search combiner doesn't use current in the way i would have expected. it only tracks metrics through current and uses the results map for the actual traces.
		//  should we change this?
		quit: func(_ *tempopb.SearchResponse) bool {
			if limit <= 0 {
				return false
			}

			return metadataCombiner.Count() >= limit
		},
	}
}

func addRootSpanNotReceivedText(results []*tempopb.TraceSearchMetadata) {
	for _, tr := range results {
		if tr.RootServiceName == "" {
			tr.RootServiceName = search.RootSpanNotYetReceivedText
		}
	}
}

func NewTypedSearch(limit int) GRPCCombiner[*tempopb.SearchResponse] {
	return NewSearch(limit).(GRPCCombiner[*tempopb.SearchResponse])
}
