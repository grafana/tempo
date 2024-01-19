package pipeline

import "net/http"

// MergeMiddlewares takes a set of ordered middlewares and merges them into a pipeline
//
//	jpe - add a bridge middleware
func Build(asyncMW []AsyncMiddleware, mw []Middleware, next http.RoundTripper) AsyncRoundTripper {
	asyncPipeline := AsyncMiddlewareFunc(func(next AsyncRoundTripper) AsyncRoundTripper {
		for i := len(asyncMW) - 1; i >= 0; i-- {
			next = asyncMW[i].Wrap(next)
		}
		return next
	})

	syncPipeline := MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		for i := len(mw) - 1; i >= 0; i-- {
			next = mw[i].Wrap(next)
		}
		return next
	})

	// bridge the two pipelines
	bridge := &pipelineBridge{
		next: syncPipeline.Wrap(next),
	}
	return asyncPipeline.Wrap(bridge)
}

var _ AsyncRoundTripper = (*pipelineBridge)(nil)

type pipelineBridge struct {
	next http.RoundTripper
}

// jpe - create a worker pool of goroutines and change req to be a channel
func (b *pipelineBridge) RoundTrip(req *http.Request) (Responses, error) {
	r, err := b.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	return NewSyncResponse(r), nil
}
