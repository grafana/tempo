package main

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-github/v93/github"
)

// retryWaitDelay is the pause between attempts. It is a variable so tests can
// zero it; production always uses one second.
var retryWaitDelay = time.Second

// retryableCall runs fn up to maxRequestAttempts times with a retryWaitDelay
// pause between attempts, for transient 5xx responses and transport failures.
// Callers use it only around idempotent operations (GET, PATCH, DELETE);
// POSTs run exactly once, so their failure semantics stay unambiguous.
func retryableCall(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 1; ; attempt++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		err = fn()
		if err == nil || attempt == maxRequestAttempts || !retryableError(err) {
			return err
		}
		if err := sleepContext(ctx, retryWaitDelay); err != nil {
			return err
		}
	}
}

// retryableError reports whether a failed call is worth replaying.
func retryableError(err error) bool {
	var statusError *github.ErrorResponse
	if errors.As(err, &statusError) {
		return statusError.Response.StatusCode >= 500
	}
	// Rate-limit responses are handled by go-github itself (which sleeps on
	// Retry-After); retrying them here would add a second, competing backoff.
	var rateLimit *github.RateLimitError
	var abuse *github.AbuseRateLimitError
	if errors.As(err, &rateLimit) || errors.As(err, &abuse) {
		return false
	}
	// Anything else without a response (DNS, connection refused, reset) is a
	// transport failure and safe to retry.
	return true
}

// sleepContext waits for the duration, returning early when the context is
// canceled so a SIGTERM can abort the retry loop.
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
