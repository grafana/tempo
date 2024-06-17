package pipeline

import (
	"github.com/grafana/tempo/pkg/api"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type traceQueryFilterWare struct {
	next    http.RoundTripper
	filters []*regexp.Regexp
}

func NewTraceQueryFilterWare(next http.RoundTripper) http.RoundTripper {
	return &traceQueryFilterWare{
		next: next,
	}
}

func NewTraceQueryFilterWareWithDenyList(denyList []string) Middleware {
	filter := make([]*regexp.Regexp, len(denyList)+1)
	for i := range denyList {
		exp, err := regexp.Compile(denyList[i])
		if err == nil {
			filter[i] = exp
		}
	}

	filter[0], _ = regexp.Compile("start")

	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return traceQueryFilterWare{
			next:    next,
			filters: filter,
		}
	})
}

func (c traceQueryFilterWare) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	//Better way to do this?
	if len(c.filters) == 0 {
		return resp, nil
	}
	//need wait group
	u, err := api.ParseSearchRequest(req)
	if err != nil {
		return resp, err
	}
	match := make(chan bool, len(c.filters))
	wg := sync.WaitGroup{}
	for range c.filters {
		wg.Add(1)
	}

	go func(qry string) {
		defer wg.Done()
		for _, re := range c.filters {
			if re.MatchString(qry) {
				match <- true
				return
			}
		}
		match <- false
	}(u.Query)

	go func() {
		wg.Wait()
		close(match)
	}()

	if <-match {

		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Body:       io.NopCloser(strings.NewReader("Query is temporarily blocked by your administrator.")),
		}, nil

	}
	return resp, nil
}
