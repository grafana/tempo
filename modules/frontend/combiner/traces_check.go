package combiner

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

var _ GRPCCombiner[*tempopb.TracesCheckResponse] = (*genericCombiner[*tempopb.TracesCheckResponse])(nil)

// NewTracesCheck returns a traces check combiner for batch processing
func NewTracesCheck(logger log.Logger) Combiner {
	c := &genericCombiner[*tempopb.TracesCheckResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.TracesCheckResponse { return &tempopb.TracesCheckResponse{} },
		current:        &tempopb.TracesCheckResponse{TraceIDs: []string{}, Metrics: &tempopb.TraceByIDMetrics{}},

		combine: func(partial *tempopb.TracesCheckResponse, final *tempopb.TracesCheckResponse, resp PipelineResponse) error {
			level.Debug(logger).Log(
				"msg", "traces check combiner: combining partial response",
				"partial_trace_count", len(partial.TraceIDs),
				"partial_inspected_bytes", func() uint64 {
					if partial.Metrics != nil {
						return partial.Metrics.InspectedBytes
					}
					return 0
				}(),
				"final_trace_count_before", len(final.TraceIDs))

			if final.TraceIDs == nil {
				final.TraceIDs = []string{}
			}

			// Merge trace IDs - add any new found traces (avoid duplicates)
			existingSet := make(map[string]bool)
			for _, traceID := range final.TraceIDs {
				existingSet[traceID] = true
			}
			
			newTraceCount := 0
			for _, traceID := range partial.TraceIDs {
				if !existingSet[traceID] {
					final.TraceIDs = append(final.TraceIDs, traceID)
					newTraceCount++
				}
			}

			// Combine metrics
			if partial.Metrics != nil {
				if final.Metrics == nil {
					final.Metrics = &tempopb.TraceByIDMetrics{}
				}
				final.Metrics.InspectedBytes += partial.Metrics.InspectedBytes
			}

			level.Debug(logger).Log(
				"msg", "traces check combiner: combined partial response",
				"new_traces_added", newTraceCount,
				"final_trace_count_after", len(final.TraceIDs),
				"final_inspected_bytes", func() uint64 {
					if final.Metrics != nil {
						return final.Metrics.InspectedBytes
					}
					return 0
				}())

			return nil
		},

		finalize: func(resp *tempopb.TracesCheckResponse) (*tempopb.TracesCheckResponse, error) {
			level.Debug(logger).Log(
				"msg", "traces check combiner: finalizing response",
				"total_trace_count", len(resp.TraceIDs),
				"total_inspected_bytes", func() uint64 {
					if resp.Metrics != nil {
						return resp.Metrics.InspectedBytes
					}
					return 0
				}())

			if resp.Metrics == nil {
				resp.Metrics = &tempopb.TraceByIDMetrics{}
			}
			if resp.TraceIDs == nil {
				resp.TraceIDs = []string{}
			}

			level.Debug(logger).Log(
				"msg", "traces check combiner: finalized response",
				"final_trace_count", len(resp.TraceIDs),
				"final_inspected_bytes", resp.Metrics.InspectedBytes)

			return resp, nil
		},

		quit: func(resp *tempopb.TracesCheckResponse) bool {
			// For now, don't early exit - let all sources respond
			// This ensures we get complete metrics and don't miss any traces
			return false
		},
	}

	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}
