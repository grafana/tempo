package pipeline

import (
	"net/http"
)

type statusCodeAdjustWare struct {
	next        http.RoundTripper
	allowedCode int
}

// NewStatusCodeAdjustWare exists to adjust status codes from the queriers for the rest of the pipeline to consume.
// This is necessary because the queriers may return 4xx status codes for bad requests, but we want to represent
// these as 500s to the rest of the pipeline. This also allows the rest of the pipeline to return 4xxs that can be trusted.
func NewStatusCodeAdjustWare() Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return statusCodeAdjustWare{
			next: next,
		}
	})
}

func NewStatusCodeAdjustWareWithAllowedCode(code int) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return statusCodeAdjustWare{
			next:        next,
			allowedCode: code,
		}
	})
}

// RoundTrip implements http.RoundTripper
func (c statusCodeAdjustWare) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if c.allowedCode != 0 && resp.StatusCode == c.allowedCode {
		return resp, nil
	}

	// adjust the response based on the following rules. any 4xx will be converted to 500.
	// if the frontend issues a bad request then externally we need to represent that as an
	// internal error
	// exceptions
	//   429 - too many requests
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
		resp.StatusCode = http.StatusInternalServerError
		resp.Status = http.StatusText(http.StatusInternalServerError)
		// leave the body alone. it will preserve the original error message
	}

	return resp, nil
}
