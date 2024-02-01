package pipeline

import (
	"context"
	"net/http"

	"github.com/grafana/tempo/modules/frontend/combiner"
)

type httpCollector struct {
	next     AsyncRoundTripper[*http.Response]
	combiner combiner.Combiner
}

// todo: long term this should return an http.Handler instead of a RoundTripper? that way it can completely
//  encapsulate all the responsibilities of converting a pipeline http.Response and error into an http.Response
//  to be

// NewHTTPCollector returns a new http collector
func NewHTTPCollector(next AsyncRoundTripper[*http.Response], combiner combiner.Combiner) http.RoundTripper {
	return httpCollector{
		next:     next,
		combiner: combiner,
	}
}

// Handle
func (r httpCollector) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	resps, err := r.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	for {
		resp, err, done := resps.Next(ctx)
		if err != nil {
			return nil, err
		}

		if resp != nil {
			err := r.combiner.AddRequest(resp, "")
			if err != nil {
				return nil, err
			}
		}

		if r.combiner.ShouldQuit() {
			break
		}

		if done {
			break
		}
	}

	resp, err := r.combiner.HTTPFinal()
	return resp, err
}
