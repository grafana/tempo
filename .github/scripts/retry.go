package main

import (
	"context"
	"net/http"
	"time"
)

// sleepContext sleeps for the duration or returns early when the context is
// canceled, so callers can abort backoff from the deadline or SIGTERM handler.
func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryingRoundTripper re-runs idempotent requests on transient failures:
// transport (network) errors and 5xx responses are retried up to maxAttempts
// in total, sleeping the exponential backoff between attempts. POST and other
// non-idempotent methods run exactly once. Requests are cloned per attempt so
// bodies can be replayed, and context cancellation aborts the retry loop.
type retryingRoundTripper struct {
	base        http.RoundTripper
	maxAttempts int
	backoff     func(attempt int) time.Duration
}

// idempotentMethod reports whether a failed request can safely be replayed.
func idempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (r *retryingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !idempotentMethod(req.Method) {
		return r.base.RoundTrip(req)
	}
	maxAttempts := r.maxAttempts
	if maxAttempts <= 0 {
		maxAttempts = maxRequestAttempts
	}
	backoff := r.backoff
	if backoff == nil {
		backoff = retryDelay
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptReq := req.Clone(req.Context())
		if req.Body != nil && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attemptReq.Body = body
		}
		resp, err := r.base.RoundTrip(attemptReq)
		if err != nil {
			lastErr = err
		} else if resp.StatusCode < 500 || attempt == maxAttempts {
			return resp, nil
		} else {
			if err := sleepContext(req.Context(), backoff(attempt)); err != nil {
				resp.Body.Close()
				return nil, err
			}
			resp.Body.Close()
			continue
		}
		if attempt == maxAttempts {
			break
		}
		if err := sleepContext(req.Context(), backoff(attempt)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}
