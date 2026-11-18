package main

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/stretchr/testify/require"
)

// zeroRetryDelay makes the one-second pauses instant for tests.
func zeroRetryDelay(t *testing.T) {
	t.Helper()
	old := retryWaitDelay
	retryWaitDelay = 0
	t.Cleanup(func() { retryWaitDelay = old })
}

func TestRetryableCallRetries5xxAndSucceeds(t *testing.T) {
	zeroRetryDelay(t)
	calls := 0
	err := retryableCall(context.Background(), func() error {
		calls++
		if calls < maxRequestAttempts {
			return &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, maxRequestAttempts, calls)
}

func TestRetryableCallGivesUpAfterThreeAttempts(t *testing.T) {
	zeroRetryDelay(t)
	calls := 0
	failed := &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadGateway}}
	err := retryableCall(context.Background(), func() error {
		calls++
		return failed
	})
	require.ErrorIs(t, err, failed)
	require.Equal(t, maxRequestAttempts, calls)
}

func TestRetryableCallDoesNotRetryRateLimitOrClientErrors(t *testing.T) {
	zeroRetryDelay(t)
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "primary rate limit", err: &github.RateLimitError{Response: &http.Response{StatusCode: http.StatusForbidden}}},
		{name: "abuse rate limit", err: &github.AbuseRateLimitError{Response: &http.Response{StatusCode: http.StatusTooManyRequests}}},
		{name: "client error", err: &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusUnprocessableEntity}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			err := retryableCall(context.Background(), func() error {
				calls++
				return test.err
			})
			require.Error(t, err)
			require.Equal(t, 1, calls)
		})
	}
}

func TestRetryableCallStopsOnCanceledContext(t *testing.T) {
	zeroRetryDelay(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := retryableCall(ctx, func() error {
		calls++
		return errors.New("unreachable")
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, calls)
}
