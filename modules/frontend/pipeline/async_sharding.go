package pipeline

import (
	"context"
	"net/http"
	"sync"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
)

type waitGroup interface {
	Add(int)
	Done()
	Wait()
}

// NewAsyncSharderFunc creates a new AsyncResponse that shards requests to the next AsyncRoundTripper[combiner.PipelineResponse]. It creates one
// goroutine per concurrent request.
func NewAsyncSharderFunc(ctx context.Context, concurrentReqs, totalReqs int, reqFn func(i int) *http.Request, next AsyncRoundTripper[combiner.PipelineResponse]) Responses[combiner.PipelineResponse] {
	var wg waitGroup
	if concurrentReqs <= 0 {
		wg = &sync.WaitGroup{}
	} else {
		bwg := boundedwaitgroup.New(uint(concurrentReqs))
		wg = &bwg
	}
	asyncResp := newAsyncResponse()

	go func() {
		defer asyncResp.SendComplete()

		for i := 0; i < totalReqs; i++ {
			req := reqFn(i)
			// else check for a request to pass down the pipeline
			if req == nil {
				continue
			}

			if err := req.Context().Err(); err != nil {
				asyncResp.SendError(err)
				continue
			}

			wg.Add(1)
			go func(r *http.Request) {
				defer wg.Done()

				resp, err := next.RoundTrip(r)
				if err != nil {
					asyncResp.SendError(err)
					return
				}

				asyncResp.Send(ctx, resp)
			}(req)
		}

		wg.Wait()
	}()

	return asyncResp
}

// NewAsyncSharderChan creates a new AsyncResponse that shards requests to the next AsyncRoundTripper[combiner.PipelineResponse] using a limited number of goroutines.
func NewAsyncSharderChan(ctx context.Context, concurrentReqs int, reqs <-chan *http.Request, resps Responses[combiner.PipelineResponse], next AsyncRoundTripper[combiner.PipelineResponse]) Responses[combiner.PipelineResponse] {
	if concurrentReqs == 0 {
		panic("NewAsyncSharderChan: concurrentReqs must be greater than 0")
	}

	wg := &sync.WaitGroup{}
	asyncResp := newAsyncResponse()

	for i := 0; i < concurrentReqs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range reqs {
				if err := req.Context().Err(); err != nil {
					asyncResp.SendError(err)
					continue
				}

				resp, err := next.RoundTrip(req)
				if err != nil {
					asyncResp.SendError(err)
					continue
				}

				asyncResp.Send(ctx, resp)
			}
		}()
	}

	go func() {
		// send any responses back the caller would like to send
		if resps != nil {
			asyncResp.Send(ctx, resps)
		}

		wg.Wait()
		asyncResp.SendComplete()
	}()

	return asyncResp
}
