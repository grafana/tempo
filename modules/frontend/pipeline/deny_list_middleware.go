package pipeline

import (
	"fmt"
	"regexp"

	"github.com/grafana/tempo/modules/frontend/combiner"
)

type urlDenylistWare struct {
	denyList []*regexp.Regexp
	next     AsyncRoundTripper[combiner.PipelineResponse]
}

func NewURLDenyListWare(denyList []string) AsyncMiddleware[combiner.PipelineResponse] {
	compiledDenylist := make([]*regexp.Regexp, 0)
	for _, v := range denyList {
		r, err := regexp.Compile(v)
		if err == nil {
			compiledDenylist = append(compiledDenylist, r)
		} else {
			panic(fmt.Sprintf("error compiling query frontend deny list regex: %s", err))
		}
	}

	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &urlDenylistWare{
			next:     next,
			denyList: compiledDenylist,
		}
	})
}

func (c urlDenylistWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	if len(c.denyList) != 0 {
		err := c.validateRequest(req.HTTPRequest().URL.String())
		if err != nil {
			return NewBadRequest(err), nil
		}
	}

	return c.next.RoundTrip(req)
}

func (c urlDenylistWare) validateRequest(url string) error {
	for _, v := range c.denyList {
		if v.MatchString(url) {
			return fmt.Errorf("Invalid request %s. This query has been identified as one that destabilizes our system. Contact your system administrator for more information", url)
		}
	}
	return nil
}
