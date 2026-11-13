package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequestRetriesOnlySafeTransientOperations(t *testing.T) {
	for _, test := range []struct {
		name        string
		method      string
		status      int
		wantCalls   int
		wantSuccess bool
	}{
		{name: "GET retries 500", method: http.MethodGet, status: http.StatusInternalServerError, wantCalls: 3, wantSuccess: true},
		{name: "GET retries 429", method: http.MethodGet, status: http.StatusTooManyRequests, wantCalls: 3, wantSuccess: true},
		{name: "DELETE retries 500", method: http.MethodDelete, status: http.StatusInternalServerError, wantCalls: 3, wantSuccess: true},
		{name: "GET does not retry 404", method: http.MethodGet, status: http.StatusNotFound, wantCalls: 1},
		{name: "POST does not retry 500", method: http.MethodPost, status: http.StatusInternalServerError, wantCalls: 1},
		{name: "POST does not retry 429", method: http.MethodPost, status: http.StatusTooManyRequests, wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls < test.wantCalls {
					w.Header().Set("Retry-After", "0")
					http.Error(w, "transient", test.status)
					return
				}
				if test.wantSuccess {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"ok":true}`)
				} else {
					http.Error(w, "failed", test.status)
				}
			}))
			defer server.Close()
			client := newGitHubClient("test")
			client.baseURL = server.URL
			client.retryDelay = func(int) time.Duration { return 0 }
			var result struct {
				OK bool `json:"ok"`
			}
			_, err := client.request(context.Background(), test.method, "/resource", nil, nil, &result)
			require.Equal(t, test.wantCalls, calls)
			if test.wantSuccess {
				require.NoError(t, err)
				require.True(t, result.OK)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestRequestDoesNotRetryInvalidJSON(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		fmt.Fprint(w, `{`)
	}))
	defer server.Close()
	client := newGitHubClient("test")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	_, err := client.request(context.Background(), http.MethodGet, "/resource", nil, nil, &map[string]any{})
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestPaginationRejectsRepeatedNextURL(t *testing.T) {
	calls := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Link", fmt.Sprintf(`<%s/items>; rel="next"`, server.URL))
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()
	client := newGitHubClient("test")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	_, err := paginateList[map[string]any](context.Background(), client, "/items", nil)
	require.ErrorContains(t, err, "repeated next URL")
	require.Equal(t, 2, calls)
}

func TestPaginationRejectsExcessivePages(t *testing.T) {
	calls := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Link", fmt.Sprintf(`<%s/items?page=%d>; rel="next"`, server.URL, calls+1))
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()
	client := newGitHubClient("test")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	_, err := paginateList[map[string]any](context.Background(), client, "/items", nil)
	require.ErrorContains(t, err, "exceeded 100 pages")
	require.Equal(t, maxPaginationPages, calls)
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)
	headers := http.Header{"Retry-After": {"12"}}
	require.Equal(t, 12*time.Second, parseRetryAfter(headers, now))
	headers.Set("Retry-After", now.Add(20*time.Second).Format(http.TimeFormat))
	require.Equal(t, 20*time.Second, parseRetryAfter(headers, now))
	headers.Set("Retry-After", "invalid")
	require.Zero(t, parseRetryAfter(headers, now))
}

func TestNextPage(t *testing.T) {
	require.Equal(t, "https://api.github.com/page=2", nextPage(`<https://api.github.com/page=2>; rel="next", <https://api.github.com/page=4>; rel="last"`))
	require.Empty(t, nextPage(`<https://api.github.com/page=4>; rel="last"`))
}
