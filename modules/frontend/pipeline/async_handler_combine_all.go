package pipeline

import (
	"context"
	"net/http"

	"github.com/grafana/tempo/modules/frontend/combiner"
)

// jpe - need?

func newCombineAllHandler(c combiner.Combiner) AsyncMiddleware { // jpe this name is terrible
	return AsyncMiddlewareFunc(func(next AsyncRoundTripper) AsyncRoundTripper {
		return combineAllHandler{
			next: next,
			c:    c,
		}
	})
}

type combineAllHandler struct {
	next AsyncRoundTripper
	c    combiner.Combiner
}

// Handle
func (r combineAllHandler) RoundTrip(req *http.Request) (Responses, error) { // jpe - this should be a standard http handler? to be added to an http endpoint?
	ctx := req.Context()
	ctx, cancel := context.WithCancel(ctx) // create a new context with a cancel function
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
			err := r.c.AddRequest(resp, "") // jpe - remove tenant!
			if err != nil {
				return nil, err // jpe format better
			}
		}

		if r.c.ShouldQuit() {
			break
		}

		if done {
			break
		}
	}

	resp, err := r.c.Complete()
	return NewSyncResponse(resp), err
}
