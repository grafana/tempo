package pipeline

import (
	"fmt"
	"net/url"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/traceql"
)

type queryValidatorWare struct {
	next              AsyncRoundTripper[combiner.PipelineResponse]
	maxQuerySizeBytes int
}

func NewQueryValidatorWare(maxQuerySizeBytes int) AsyncMiddleware[combiner.PipelineResponse] {
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &queryValidatorWare{
			next:              next,
			maxQuerySizeBytes: maxQuerySizeBytes,
		}
	})
}

func (c queryValidatorWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	query := req.HTTPRequest().URL.Query()
	err := c.validateTraceQLQuery(query)
	if err != nil {
		return NewBadRequest(err), nil
	}
	return c.next.RoundTrip(req)
}

func (c queryValidatorWare) validateTraceQLQuery(queryParams url.Values) error {
	if err := ValidateTraceQLQueryParamsSize(queryParams, c.maxQuerySizeBytes); err != nil {
		return err
	}

	traceQLQuery := traceQLQueryFromParams(queryParams)
	if traceQLQuery != "" {
		expr, err := traceql.ParseNoOptimizations(traceQLQuery)
		if err == nil {
			err = traceql.Validate(expr)
		}
		if err != nil {
			return fmt.Errorf("invalid TraceQL query: %w", err)
		}
	}
	return nil
}

func ValidateTraceQLQueryParamsSize(queryParams url.Values, maxQuerySizeBytes int) error {
	for _, param := range []string{"q", "query"} {
		for _, traceQLQuery := range queryParams[param] {
			if err := ValidateTraceQLQuerySize(traceQLQuery, maxQuerySizeBytes); err != nil {
				return err
			}
		}
	}
	return nil
}

func ValidateTraceQLQuerySize(traceQLQuery string, maxQuerySizeBytes int) error {
	// Reject huge queries before parsing to avoid parser CPU and memory exhaustion.
	if traceQLQuery != "" && len(traceQLQuery) > maxQuerySizeBytes {
		return fmt.Errorf("TraceQL expression exceeds the configured maximum size of %d bytes, reduce the query expression size or contact your system administrator", maxQuerySizeBytes)
	}
	return nil
}

func traceQLQueryFromParams(queryParams url.Values) string {
	var traceQLQuery string
	if queryParams.Has("q") {
		traceQLQuery = queryParams.Get("q")
	}
	if queryParams.Has("query") {
		traceQLQuery = queryParams.Get("query")
	}
	return traceQLQuery
}
