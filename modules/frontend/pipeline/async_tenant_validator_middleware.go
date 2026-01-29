package pipeline

import (
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/validation"
)

type tenantValidatorRoundTripper struct {
	next AsyncRoundTripper[combiner.PipelineResponse]
}

// NewTenantValidatorMiddleware returns a middleware that validates the tenant ID in requests.
// It assumes there's only one tenant ID per request, otherwise it returns an error.
// For this reason it must run after the multiTenantMiddleware.
func NewTenantValidatorMiddleware() AsyncMiddleware[combiner.PipelineResponse] {
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &tenantValidatorRoundTripper{next: next}
	})
}

func (t *tenantValidatorRoundTripper) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	_, err := validation.ExtractValidTenantID(req.Context())
	if err != nil {
		return NewBadRequest(err), nil
	}
	return t.next.RoundTrip(req)
}
