package pipeline

import (
	"github.com/grafana/tempo/modules/frontend/combiner"
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
