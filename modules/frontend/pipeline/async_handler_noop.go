package pipeline

import (
	"github.com/grafana/tempo/v2/modules/frontend/combiner"
)

// NewNoopMiddleware returns a middleware that is a passthrough only
func NewNoopMiddleware() AsyncMiddleware[combiner.PipelineResponse] {
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return AsyncRoundTripperFunc[combiner.PipelineResponse](func(req Request) (Responses[combiner.PipelineResponse], error) {
			return next.RoundTrip(req)
		})
	})
}
