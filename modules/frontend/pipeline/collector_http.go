package pipeline

import (
	"context"
	"net/http"
	"sync"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"go.uber.org/atomic"
)

type httpCollector struct {
	next     AsyncRoundTripper[combiner.PipelineResponse]
	combiner combiner.Combiner
}

// todo: long term this should return an http.Handler instead of a RoundTripper? that way it can completely
//  encapsulate all the responsibilities of converting a pipeline http.Response and error into an http.Response
//  to be

// NewHTTPCollector returns a new http collector
func NewHTTPCollector(next AsyncRoundTripper[combiner.PipelineResponse], combiner combiner.Combiner) http.RoundTripper {
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

	err = addNextAsync(ctx, resps, r.next, r.combiner, nil)
	if err != nil {
		return nil, err
	}

	return r.combiner.HTTPFinal()
}

func addNextAsync(ctx context.Context, resps Responses[combiner.PipelineResponse], next AsyncRoundTripper[combiner.PipelineResponse], c combiner.Combiner, callback func() error) error {
	respChan := make(chan combiner.PipelineResponse)
	overallErr := atomic.Error{}
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for resp := range respChan {
				err := c.AddResponse(resp)
				if err != nil {
					overallErr.Store(err)
				}
			}
		}()
	}

	for {
		if ctx.Err() != nil {
			overallErr.Store(ctx.Err())
			break
		}

		resp, done, err := resps.Next(ctx)
		if err != nil {
			overallErr.Store(err)
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
				overallErr.Store(err)
				break
			}
		}
	}

	close(respChan)
	wg.Wait()

	return overallErr.Load()
}
