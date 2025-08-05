package combiner

import (
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

var _ GRPCCombiner[*tempopb.TraceLookupResponse] = (*genericCombiner[*tempopb.TraceLookupResponse])(nil)

// NewTraceLookup returns a trace lookup combiner
func NewTraceLookup() Combiner {
	metricsCombiner := NewSearchMetricsCombiner()

	c := &genericCombiner[*tempopb.TraceLookupResponse]{
		httpStatusCode: 200,
		new:            func() *tempopb.TraceLookupResponse { return &tempopb.TraceLookupResponse{} },
		current:        &tempopb.TraceLookupResponse{Results: make(map[string]bool), Metrics: &tempopb.SearchMetrics{}},

		combine: func(partial *tempopb.TraceLookupResponse, final *tempopb.TraceLookupResponse, resp PipelineResponse) error {
			if final.Results == nil {
				final.Results = make(map[string]bool)
			}

			// Merge the results - if any partial response indicates a trace exists, mark it as found
			for traceID, exists := range partial.Results {
				if exists || final.Results[traceID] == false {
					final.Results[traceID] = exists
				}
			}

			// Combine metrics
			var metrics *tempopb.SearchMetrics
			if partial.Metrics != nil {
				metrics = partial.Metrics
			} else {
				metrics = &tempopb.SearchMetrics{}
			}
			metricsCombiner.Combine(metrics, resp)

			return nil
		},

		finalize: func(resp *tempopb.TraceLookupResponse) (*tempopb.TraceLookupResponse, error) {
			if resp.Metrics == nil {
				resp.Metrics = &tempopb.SearchMetrics{}
			}
			resp.Metrics = metricsCombiner.Metrics
			return resp, nil
		},

		quit: func(*tempopb.TraceLookupResponse) bool {
			return false
		},
	}

	initHTTPCombiner(c, api.HeaderAcceptJSON)
	return c
}