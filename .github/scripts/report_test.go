package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// reportStub records the requests a reportFailure test server received.
type reportStub struct {
	mu       sync.Mutex
	requests []string
}

func (s *reportStub) record(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, r.Method+" "+r.URL.Path)
}

func (s *reportStub) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func TestReportFailureCreatesThenReusesIssue(t *testing.T) {
	stub := &reportStub{}
	searches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.record(r)
		switch {
		case r.Method == http.MethodGet:
			searches++
			if searches == 1 {
				fmt.Fprint(w, `{"total_count":0,"items":[]}`)
			} else {
				fmt.Fprint(w, `{"total_count":1,"items":[{"number":42}]}`)
			}
		case strings.HasSuffix(r.URL.Path, "/issues"):
			fmt.Fprint(w, `{"number":100}`)
		case strings.HasSuffix(r.URL.Path, "/comments"):
			fmt.Fprint(w, `{"id":1}`)
		default:
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := testGitHubClient(t, server.URL)
	const runURL = "https://example.invalid/run"
	require.NoError(t, reportFailure(context.Background(), client, "grafana/tempo", runURL))
	require.NoError(t, reportFailure(context.Background(), client, "grafana/tempo", runURL))
	require.Equal(t, []string{
		"GET /search/issues",
		"POST /repos/grafana/tempo/issues",
		"GET /search/issues",
		"POST /repos/grafana/tempo/issues/42/comments",
	}, stub.snapshot())
}

func TestReportFailureSearchErrorDoesNotCreateDuplicate(t *testing.T) {
	zeroRetryDelay(t)
	stub := &reportStub{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.record(r)
		http.Error(w, "search unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()
	err := reportFailure(context.Background(), testGitHubClient(t, server.URL), "grafana/tempo", "https://example.invalid/run")
	require.ErrorContains(t, err, "find existing sweep failure issue")
	// The search is retried three times, but a duplicate issue is never created.
	require.Equal(t, []string{"GET /search/issues", "GET /search/issues", "GET /search/issues"}, stub.snapshot())
}

func TestReportFailureSurfacesCommentAndCreateFailures(t *testing.T) {
	for _, test := range []struct {
		name     string
		items    string
		failPath string
		wantErr  string
	}{
		{name: "create failure", items: `{"total_count":0,"items":[]}`, failPath: "/issues", wantErr: "create sweep failure issue"},
		{name: "comment failure", items: `{"total_count":1,"items":[{"number":42}]}`, failPath: "/comments", wantErr: "comment on sweep failure issue #42"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet:
					fmt.Fprint(w, test.items)
				case strings.HasSuffix(r.URL.Path, test.failPath):
					http.Error(w, "write failed", http.StatusInternalServerError)
				default:
					http.Error(w, "unexpected", http.StatusNotFound)
				}
			}))
			defer server.Close()
			err := reportFailure(context.Background(), testGitHubClient(t, server.URL), "grafana/tempo", "https://example.invalid/run")
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}
