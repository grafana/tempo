package pipeline

import (
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/validation"
)

type tenantValidatorRoundTripper struct {
	next AsyncRoundTripper[combiner.PipelineResponse]
}

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
