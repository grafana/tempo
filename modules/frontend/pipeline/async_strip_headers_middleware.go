package pipeline

import (
	"net/http"
	"slices"

	"github.com/grafana/tempo/v3/modules/frontend/combiner"
)

type stripHeadersWare struct {
	allowed map[string]struct{}
	next    AsyncRoundTripper[combiner.PipelineResponse]
}

// NewStripHeadersWare creates a middleware that strips headers not in the allow list. This exists to reduce allocations further
// down the pipeline. All request headers should be handled at the Combiner/Collector levels. Once the request is in the pipeline
// nothing else needs HTTP headers. Stripping them out reduces allocations for copying, marshalling and unmashalling them to sometimes
// 100s of thousands of subrequests.
func NewStripHeadersWare(allowList []string) AsyncMiddleware[combiner.PipelineResponse] {
	// build allowed map
	allowed := make(map[string]struct{}, len(allowList))
	for _, header := range allowList {
		allowed[http.CanonicalHeaderKey(header)] = struct{}{}
	}

	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &stripHeadersWare{
			next:    next,
			allowed: allowed,
		}
	})
}

func (c stripHeadersWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	httpReq := req.HTTPRequest()

	// The caller reads the original headers after the pipeline returns, e.g. for log_query_request_headers,
	// so the stripped headers go on a shallow copy of the request.
	stripped := make(http.Header, len(c.allowed))
	for header, values := range httpReq.Header {
		if _, ok := c.allowed[http.CanonicalHeaderKey(header)]; ok {
			stripped[header] = slices.Clone(values)
		}
	}

	strippedReq := httpReq.WithContext(httpReq.Context())
	strippedReq.Header = stripped

	return c.next.RoundTrip(req.CloneFromHTTPRequest(strippedReq))
}
