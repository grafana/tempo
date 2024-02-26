package pipeline

import (
	"net/http"
	"sync"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
)

type waitGroup interface {
	Add(int)
	Done()
	Wait()
}

// NewAsyncSharder creates a new AsyncResponse that shards requests to the next AsyncRoundTripper[*http.Response]. It creates one
// goroutine per concurrent request.
func NewAsyncSharder(concurrentReqs, totalReqs int, reqFn func(i int) *http.Request, next AsyncRoundTripper[*http.Response]) Responses[*http.Response] {
	var wg waitGroup
	if concurrentReqs <= 0 {
		wg = &sync.WaitGroup{}
	} else {
		bwg := boundedwaitgroup.New(uint(concurrentReqs))
		wg = &bwg
	}
	asyncResp := newAsyncResponse()

	go func() {
		defer asyncResp.done()

		for i := 0; i < totalReqs; i++ {
			req := reqFn(i)
			// else check for a request to pass down the pipeline
			if req == nil {
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

				asyncResp.Send(resp)
			}(req)
		}

		wg.Wait()
	}()

	return asyncResp
}

// NewAsyncSharderLimitedGoroutines creates a new AsyncResponse that shards requests to the next AsyncRoundTripper[*http.Response] using a limited number of goroutines.
func NewAsyncSharderLimitedGoroutines(concurrent int, reqs <-chan *http.Request, resps Responses[*http.Response], next AsyncRoundTripper[*http.Response]) Responses[*http.Response] {
	asyncResp := newAsyncResponse()
	concurrencyLimiter := boundedwaitgroup.New(uint(concurrent))

	const minLimitedGoroutines = 100

	// todo: concurrent/5 is a very roughly estimated number. make this configurable? tune it?
	limitedRoutines := max(minLimitedGoroutines, concurrent/5)
	goroutines := min(limitedRoutines, concurrent)

	wg := sync.WaitGroup{}
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for req := range reqs {
				concurrencyLimiter.Add(1)

				resp, err := next.RoundTrip(req)
				if err != nil {
					asyncResp.SendError(err)
					concurrencyLimiter.Done()
					continue
				}

				asyncResp.Send(resp)
				concurrencyLimiter.Done()
			}
		}()
	}

	go func() {
		if resps != nil {
			asyncResp.Send(resps)
		}

		wg.Wait()
		asyncResp.done()
	}()

	return asyncResp
}
