package pipeline

import (
	"context"
	"net/http"
	"sync"

	"go.uber.org/atomic"

	"github.com/grafana/tempo/modules/frontend/combiner"
)

type httpCollector struct {
	next      AsyncRoundTripper[combiner.PipelineResponse]
	combiner  combiner.Combiner
	consumers int
}

// todo: long term this should return an http.Handler instead of a RoundTripper? that way it can completely
//  encapsulate all the responsibilities of converting a pipeline http.Response and error into an http.Response
//  to be

// NewHTTPCollector returns a new http collector
func NewHTTPCollector(next AsyncRoundTripper[combiner.PipelineResponse], consumers int, combiner combiner.Combiner) http.RoundTripper {
	return httpCollector{
		next:      next,
		combiner:  combiner,
		consumers: consumers,
	}
}

// RoundTrip implements the http.RoundTripper interface
func (r httpCollector) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	ctx, span := tracer.Start(ctx, "httpCollector.RoundTrip")
	defer span.End()

	req = req.WithContext(ctx)

	resps, err := r.next.RoundTrip(NewHTTPRequest(req))
	if err != nil {
		return nil, err
	}
	span.AddEvent("next.RoundTrip done")

	err = consumeAndCombineResponses(ctx, r.consumers, resps, r.combiner, nil)
	if err != nil {
		return nil, err
	}
	span.AddEvent("consumeAndCombineResponses done")

	resp, err := r.combiner.HTTPFinal()
	if err != nil {
		return nil, err
	}
	span.AddEvent("combiner.HTTPFinal done")

	// we don't get context cancellation errors from the HTTPFinal,
	// so we need to check, and return to downstream callers
	if req.Context().Err() != nil {
		return nil, req.Context().Err()
	}
	return resp, err
}

func consumeAndCombineResponses(ctx context.Context, consumers int, resps Responses[combiner.PipelineResponse], c combiner.Combiner, callback func() error) error {
	respChan := make(chan combiner.PipelineResponse)
	overallErr := atomic.Error{}
	wg := sync.WaitGroup{}

	setErr := func(err error) {
		overallErr.CompareAndSwap(nil, err)
	}

	if consumers <= 0 {
		consumers = 10
	}

	for i := 0; i < consumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for resp := range respChan {
				err := c.AddResponse(resp)
				if err != nil {
					setErr(err)
				}
			}
		}()
	}

	for {
		if ctx.Err() != nil {
			setErr(ctx.Err())
			break
		}

		resp, done, err := resps.Next(ctx)
		if err != nil {
			setErr(err)
			break
		}

		if resp != nil {
			respChan <- resp
		}

		if overallErr.Load() != nil {
			break
		}

		if c.ShouldQuit() {
			break
		}

		if done {
			break
		}

		if callback != nil {
			err = callback()
			if err != nil {
				setErr(err)
				break
			}
		}
	}

	close(respChan)
	wg.Wait()

	return overallErr.Load()
}
