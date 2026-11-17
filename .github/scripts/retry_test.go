package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newRetryTestTransport() *retryingRoundTripper {
	return &retryingRoundTripper{base: http.DefaultTransport, maxAttempts: maxRequestAttempts, backoff: func(int) time.Duration { return 0 }}
}

func TestRetryingRoundTripperRetriesIdempotent5xx(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < maxRequestAttempts {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()
	client := &http.Client{Transport: newRetryTestTransport()}
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, maxRequestAttempts, calls)
}

func TestRetryingRoundTripperDoesNotRetryPost(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()
	client := &http.Client{Transport: newRetryTestTransport()}
	resp, err := client.Post(server.URL, "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	require.Equal(t, 1, calls)
}

func TestRetryingRoundTripperGivesUpAndReturnsLastResponse(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := &http.Client{Transport: newRetryTestTransport()}
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	require.Equal(t, maxRequestAttempts, calls)
}

func TestRetryingRoundTripperReplaysRequestBody(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		buf := make([]byte, 4)
		for {
			n, err := r.Body.Read(buf)
			b.Write(buf[:n])
			if err != nil {
				break
			}
		}
		bodies = append(bodies, b.String())
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()
	client := &http.Client{Transport: newRetryTestTransport()}
	req, err := http.NewRequest(http.MethodPatch, server.URL, strings.NewReader(`{"a":1}`))
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, maxRequestAttempts, len(bodies))
	for _, body := range bodies {
		require.Equal(t, `{"a":1}`, body)
	}
}

func TestRetryingRoundTripperCancellationStopsBackoff(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	transport := &retryingRoundTripper{base: http.DefaultTransport, maxAttempts: maxRequestAttempts, backoff: func(int) time.Duration {
		cancel()
		return time.Hour
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	_, err = transport.RoundTrip(req)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
}
