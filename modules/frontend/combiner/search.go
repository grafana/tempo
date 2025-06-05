package combiner

import (
	"net/http"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

var _ PipelineResponse = (*SearchJobResponse)(nil)

type SearchShards struct {
	TotalJobs               uint32
	CompletedThroughSeconds uint32
}

type SearchJobResponse struct {
	TotalBlocks int
	TotalJobs   int
	TotalBytes  uint64
	Shards      []SearchShards
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

var _ GRPCCombiner[*tempopb.SearchResponse] = (*genericCombiner[*tempopb.SearchResponse])(nil)

// NewSearch returns a search combiner
func NewSearch(limit int, keepMostRecent bool) Combiner {
	metadataCombiner := traceql.NewMetadataCombiner(limit, keepMostRecent)
	diffTraces := map[string]struct{}{}
	completedThroughTracker := &ShardCompletionTracker{}
	metricsCombiner := NewSearchMetricsCombiner()

	c := &genericCombiner[*tempopb.SearchResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.SearchResponse { return &tempopb.SearchResponse{} },
		current:        &tempopb.SearchResponse{Metrics: &tempopb.SearchMetrics{}},
		combine: func(partial *tempopb.SearchResponse, final *tempopb.SearchResponse, resp PipelineResponse) error {
			requestIdx, ok := resp.RequestData().(int)
			if ok && keepMostRecent {
				completedThroughTracker.addShardIdx(requestIdx)
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
					completedThroughTracker.addShards(sj.Shards)
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
					completedThroughSeconds := completedThroughTracker.completedThroughSeconds
					// if all jobs are completed then let's just return everything the combiner has
					if diff.Metrics.CompletedJobs == diff.Metrics.TotalJobs && diff.Metrics.TotalJobs > 0 {
						completedThroughSeconds = 1
					}

					// if we've not completed any shards, then return nothing
					if completedThroughSeconds == 0 {
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
			completedThroughSeconds := completedThroughTracker.completedThroughSeconds
			// have we completed any shards?
			if completedThroughSeconds == 0 {
				completedThroughSeconds = traceql.TimestampNever
			}

			return metadataCombiner.IsCompleteFor(completedThroughSeconds)
		},
	}
	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}

func addRootSpanNotReceivedText(results []*tempopb.TraceSearchMetadata) {
	for _, tr := range results {
		if tr.RootServiceName == "" {
			tr.RootServiceName = search.RootSpanNotYetReceivedText
		}
	}
}

func NewTypedSearch(limit int, keepMostRecent bool) GRPCCombiner[*tempopb.SearchResponse] {
	return NewSearch(limit, keepMostRecent).(GRPCCombiner[*tempopb.SearchResponse])
}

// ShardCompletionTracker
type ShardCompletionTracker struct {
	shards         []SearchShards
	foundResponses []int

	completedThroughSeconds uint32
	curShard                int
}

func (s *ShardCompletionTracker) addShards(shards []SearchShards) uint32 {
	if len(shards) == 0 {
		return s.completedThroughSeconds
	}

	s.shards = shards

	// grow foundResponses to match while keeping the existing values
	if len(s.shards) > len(s.foundResponses) {
		temp := make([]int, len(s.shards))
		copy(temp, s.foundResponses)
		s.foundResponses = temp
	}

	s.incrementCurShardIfComplete()

	return s.completedThroughSeconds
}

// Add adds a response to the tracker and returns the allowed completedThroughSeconds
func (s *ShardCompletionTracker) addShardIdx(shardIdx int) uint32 {
	// we haven't received shards yet
	if len(s.shards) == 0 {
		// if shardIdx doesn't fit in foundResponses then alloc a new slice and copy foundResponses forward
		if shardIdx >= len(s.foundResponses) {
			temp := make([]int, shardIdx+1)
			copy(temp, s.foundResponses)
			s.foundResponses = temp
		}

		// and record this idx for when we get shards
		s.foundResponses[shardIdx]++

		return 0
	}

	//
	if shardIdx >= len(s.foundResponses) {
		return s.completedThroughSeconds
	}

	s.foundResponses[shardIdx]++
	s.incrementCurShardIfComplete()

	return s.completedThroughSeconds
}

// incrementCurShardIfComplete tests to see if the current shard is complete and increments it if so.
// it does this repeatedly until it finds a shard that is not complete.
func (s *ShardCompletionTracker) incrementCurShardIfComplete() {
	for {
		if s.curShard >= len(s.shards) {
			break
		}

		if s.foundResponses[s.curShard] == int(s.shards[s.curShard].TotalJobs) {
			s.completedThroughSeconds = s.shards[s.curShard].CompletedThroughSeconds
			s.curShard++
		} else {
			break
		}
	}
}
