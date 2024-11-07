package pipeline

import (
	"github.com/grafana/tempo/modules/frontend/combiner"
)

type stripHeadersWare struct {
	allowed map[string]struct{}
	next    AsyncRoundTripper[combiner.PipelineResponse]
}

func NewStripHeadersWare(allowList []string) AsyncMiddleware[combiner.PipelineResponse] {
	// build allowed map
	allowed := make(map[string]struct{}, len(allowList))
	for _, header := range allowList {
		allowed[header] = struct{}{}
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

	if len(c.allowed) == 0 {
		clear(httpReq.Header)
	} else {
		// clear out headers not in allow list
		for header := range httpReq.Header {
			if _, ok := c.allowed[header]; !ok {
				delete(httpReq.Header, header)
			}
		}
	}

	return c.next.RoundTrip(req.CloneFromHTTPRequest(httpReq))
}
