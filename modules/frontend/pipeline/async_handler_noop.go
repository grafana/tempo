package pipeline

import (
	"net/http"
)

// NewNoopMiddleware returns a middleware that is a passthrough only
func NewNoopMiddleware() AsyncMiddleware[*http.Response] {
	return AsyncMiddlewareFunc[*http.Response](func(next AsyncRoundTripper[*http.Response]) AsyncRoundTripper[*http.Response] {
		return AsyncRoundTripperFunc[*http.Response](func(req *http.Request) (Responses[*http.Response], error) {
			return next.RoundTrip(req)
		})
	})
}
