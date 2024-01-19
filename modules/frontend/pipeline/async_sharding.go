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

// jpe reqFn returns a request and response. remove when/if we put caching in its own middleware
func NewAsyncSharder(concurrent int, reqFn func(i int) (*http.Request, *http.Response), next AsyncRoundTripper) Responses { // jpe - added i here for tenant sharder, ditch? change order to put next before reqFn?
	var wg waitGroup
	if concurrent <= 0 {
		wg = &sync.WaitGroup{}
	} else {
		bwg := boundedwaitgroup.New(uint(concurrent))
		wg = &bwg
	}
	asyncResp := newAsyncResponse()

	go func() {
		defer asyncResp.done()

		reqIdx := 0
		for {
			req, resp := reqFn(reqIdx)
			reqIdx++

			// if we have a response, likely from cache, send it back
			if resp != nil {
				asyncResp.send(NewSyncResponse(resp))
				continue
			}

			// else check for a request to pass down the pipeline
			if req == nil {
				break
			}

			// jpe pass async forward as well? instead of using go routines?
			wg.Add(1)
			go func(r *http.Request) {
				defer wg.Done()

				resp, _ := next.RoundTrip(r) // jpe - handle err
				asyncResp.send(resp)
			}(req)
		}

		wg.Wait()
	}()

	return asyncResp
}
