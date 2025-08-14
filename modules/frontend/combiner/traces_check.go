package combiner

import (
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

var _ GRPCCombiner[*tempopb.TracesCheckResponse] = (*genericCombiner[*tempopb.TracesCheckResponse])(nil)

// NewTracesCheck returns a traces check combiner for batch processing
func NewTracesCheck() Combiner {
	c := &genericCombiner[*tempopb.TracesCheckResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.TracesCheckResponse { return &tempopb.TracesCheckResponse{} },
		current:        &tempopb.TracesCheckResponse{TraceIDs: []string{}, Metrics: &tempopb.TraceByIDMetrics{}},

		combine: func(partial *tempopb.TracesCheckResponse, final *tempopb.TracesCheckResponse, resp PipelineResponse) error {
			if final.TraceIDs == nil {
				final.TraceIDs = []string{}
			}

			// Merge trace IDs - add any new found traces (avoid duplicates)
			existingSet := make(map[string]bool)
			for _, traceID := range final.TraceIDs {
				existingSet[traceID] = true
			}
			
			for _, traceID := range partial.TraceIDs {
				if !existingSet[traceID] {
					final.TraceIDs = append(final.TraceIDs, traceID)
				}
			}

			// Combine metrics
			if partial.Metrics != nil {
				if final.Metrics == nil {
					final.Metrics = &tempopb.TraceByIDMetrics{}
				}
				final.Metrics.InspectedBytes += partial.Metrics.InspectedBytes
			}

			return nil
		},

		finalize: func(resp *tempopb.TracesCheckResponse) (*tempopb.TracesCheckResponse, error) {
			if resp.Metrics == nil {
				resp.Metrics = &tempopb.TraceByIDMetrics{}
			}
			if resp.TraceIDs == nil {
				resp.TraceIDs = []string{}
			}
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
