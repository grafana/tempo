package pipeline

import (
	"net/http"
	"regexp"
)

type traceQueryFilterWare struct {
	next    http.RoundTripper
	filters []*regexp.Regexp
}

func NewTraceQueryFilterWare(denyList []*regexp.Regexp) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return traceQueryFilterWare{
			next:    next,
			filters: denyList,
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

	u := req.URL.RawQuery
	match := make(chan bool, len(c.filters))

	go func(qry string) {
		for _, re := range c.filters {
			if re.MatchString(qry) {
				match <- true
				return
			}
		}
		match <- false
	}(u)
	close(match)

	if <-match {
		resp.StatusCode = http.StatusBadRequest
		resp.Status = http.StatusText(http.StatusBadRequest)
		//Anything in the body? An error message?
	}
	return resp, nil
}
