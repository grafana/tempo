package pipeline

import "net/http"

// jpe - need?

func newNoopHandler() AsyncMiddleware {
	return AsyncMiddlewareFunc(func(next AsyncRoundTripper) AsyncRoundTripper {
		return noopHandler{
			next: next,
		}
	})
}

type noopHandler struct {
	next AsyncRoundTripper
}

// Handle
func (r noopHandler) RoundTrip(req *http.Request) (Responses, error) {
	return r.next.RoundTrip(req)
}
