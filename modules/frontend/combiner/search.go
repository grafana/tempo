package combiner

import (
	"net/http"

	"github.com/grafana/tempo/modules/frontend/shardtracker"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

// SearchJobResponse wraps shardtracker.JobMetadata and implements PipelineResponse.
type SearchJobResponse struct {
	shardtracker.JobMetadata
}

func (s *SearchJobResponse) HTTPResponse() *http.Response {
	return nil
}

func (s *SearchJobResponse) RequestData() any {
	return nil
}

func (s *SearchJobResponse) IsMetadata() bool {
	return true
}

var _ PipelineResponse = (*SearchJobResponse)(nil)

var _ GRPCCombiner[*tempopb.SearchResponse] = (*genericCombiner[*tempopb.SearchResponse])(nil)

// NewSearch returns a search combiner
func NewSearch(limit int, keepMostRecent bool, marshalingFormat api.MarshallingFormat) Combiner {
	metadataCombiner := traceql.NewMetadataCombiner(limit, keepMostRecent)
	diffTraces := map[string]struct{}{}
	completedThroughTracker := &shardtracker.CompletionTracker{}
	metricsCombiner := NewSearchMetricsCombiner()

	c := &genericCombiner[*tempopb.SearchResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchResponse { return &tempopb.SearchResponse{} },
		current:        &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.SearchResponse, final *tempopb.SearchResponse, resp PipelineResponse) error {
			requestIdx, ok := resp.RequestData().(int)
			if ok && keepMostRecent {
				completedThroughTracker.AddShardIdx(requestIdx)
			}

			for _, t := range partial.Traces {
				if metadataCombiner.AddMetadata(t) {
					// record modified traces
					diffTraces[t.TraceID] = struct{}{}
				}
			}

			metricsCombiner.Combine(partial.Metrics, resp)

			return nil
		},
		metadata: func(resp PipelineResponse, final *tempopb.SearchResponse) error {
			if sj, ok := resp.(*SearchJobResponse); ok && sj != nil {
				sjMetrics := &tempopb.SearchMetrics{
					TotalBlocks:     uint32(sj.TotalBlocks), //nolint:gosec
					TotalJobs:       uint32(sj.TotalJobs),   //nolint:gosec
					TotalBlockBytes: sj.TotalBytes,
				}
				metricsCombiner.CombineMetadata(sjMetrics, resp)

				if keepMostRecent {
					completedThroughTracker.AddShards(sj.Shards)
				}
			}

			return nil
		},
		finalize: func(final *tempopb.SearchResponse) (*tempopb.SearchResponse, error) {
			// metrics are already combined on the passed in final
			final.Traces = metadataCombiner.Metadata()
			final.Metrics = metricsCombiner.Metrics
			addRootSpanNotReceivedText(final.Traces)
			return final, nil
		},
		diff: func(current *tempopb.SearchResponse) (*tempopb.SearchResponse, error) {
			// wipe out any existing traces and recreate from the map
			diff := &tempopb.SearchResponse{
				Traces:  make([]*tempopb.TraceSearchMetadata, 0, len(diffTraces)),
				Metrics: metricsCombiner.Metrics,
			}
			metadataFn := metadataCombiner.Metadata
			if keepMostRecent {
				metadataFn = func() []*tempopb.TraceSearchMetadata {
					completedThroughSeconds := completedThroughTracker.CompletedThroughSeconds()

					// if we've not completed any shards, then return nothing
					if completedThroughSeconds == shardtracker.TimestampUnknown {
						return nil
					}

					return metadataCombiner.MetadataAfter(completedThroughSeconds)
				}
			}

			for _, tr := range metadataFn() {
				// if not in the map, skip. we haven't seen an update
				if _, ok := diffTraces[tr.TraceID]; !ok {
					continue
				}

				delete(diffTraces, tr.TraceID)
				diff.Traces = append(diff.Traces, tr)
			}

			addRootSpanNotReceivedText(diff.Traces)

			return diff, nil
		},
		quit: func(_ *tempopb.SearchResponse) bool {
			completedThroughSeconds := completedThroughTracker.CompletedThroughSeconds()
			// have we completed any shards?
			if completedThroughSeconds == shardtracker.TimestampUnknown {
				completedThroughSeconds = traceql.TimestampNever
			}

			return metadataCombiner.IsCompleteFor(completedThroughSeconds)
		},
	}
	initHTTPCombiner(c, marshalingFormat)
	return c
}

func addRootSpanNotReceivedText(results []*tempopb.TraceSearchMetadata) {
	for _, tr := range results {
		if tr.RootServiceName == "" {
			tr.RootServiceName = search.RootSpanNotYetReceivedText
		}
	}
}

func NewTypedSearch(limit int, keepMostRecent bool, marshalingFormat api.MarshallingFormat) GRPCCombiner[*tempopb.SearchResponse] {
	return NewSearch(limit, keepMostRecent, marshalingFormat).(GRPCCombiner[*tempopb.SearchResponse])
}
