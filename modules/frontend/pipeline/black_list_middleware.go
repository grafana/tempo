package pipeline

import (
	"fmt"
	"regexp"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/frontend/combiner"
)

type urlBlacklistWare struct {
	blackList []*regexp.Regexp
	next      AsyncRoundTripper[combiner.PipelineResponse]
}

func NewURLBlackListWare(blackList []string, logger log.Logger) AsyncMiddleware[combiner.PipelineResponse] {
	compiledBlacklist := make([]*regexp.Regexp, 0)
	for _, v := range blackList {
		r, err := regexp.Compile(v)
		if err == nil {
			compiledBlacklist = append(compiledBlacklist, r)
		} else {
			level.Warn(logger).Log("msg", "error compiling query frontend blacklist regex", "error", err)
		}
	}

	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &urlBlacklistWare{
			next:      next,
			blackList: compiledBlacklist,
		}
	})
}

func (c urlBlacklistWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	if len(c.blackList) != 0 {
		err := c.validateRequest(req.HTTPRequest().URL.Path)
		if err != nil {
			return NewBadRequest(err), nil
		}
	}

	return c.next.RoundTrip(req)
}

func (c urlBlacklistWare) validateRequest(url string) error {
	for _, v := range c.blackList {
		if v.MatchString(url) {
			return fmt.Errorf("Invalid request %s, URL is blacklisted", url)
		}
	}
	return nil
}
