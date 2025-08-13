package combiner

import (
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

var _ GRPCCombiner[*tempopb.TracesCheckResponse] = (*genericCombiner[*tempopb.TracesCheckResponse])(nil)

// NewTracesCheck returns a traces check combiner
func NewTracesCheck() Combiner {
	c := &genericCombiner[*tempopb.TracesCheckResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.TracesCheckResponse { return &tempopb.TracesCheckResponse{} },
		current:        &tempopb.TracesCheckResponse{Exists: false, Metrics: &tempopb.TraceByIDMetrics{}},

		combine: func(partial *tempopb.TracesCheckResponse, final *tempopb.TracesCheckResponse, resp PipelineResponse) error {
			// If any partial response indicates the trace exists, mark it as found
			if partial.Exists {
				final.Exists = true
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
			return resp, nil
		},

		quit: func(resp *tempopb.TracesCheckResponse) bool {
			// Early exit if we found the trace - no need to check more sources
			return resp.Exists
		},
	}

	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}
