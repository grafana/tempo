package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v93/github"
	"github.com/stretchr/testify/require"
)

func testConfig() config {
	cfg := defaultConfig()
	cfg.assigneePool = []string{"Alice", "Bob"}
	cfg.maxAssignments = 1
	return cfg
}

// testGitHubClient builds a go-github client pointed at a test server,
// mirroring production construction.
func testGitHubClient(t *testing.T, serverURL string) *github.Client {
	t.Helper()
	client, err := github.NewClient(
		github.WithHTTPClient(&http.Client{Timeout: apiRequestTimeout}),
		github.WithAuthToken("test-token"),
		github.WithMaxSecondaryRateLimitRetryAfterDuration(maxSecondaryRateLimitRetryTime),
		github.WithURLs(&serverURL, &serverURL),
	)
	require.NoError(t, err)
	return client
}

func testPull(number int) pullRequest {
	return pullRequest{
		Number:    number,
		CreatedAt: testNow.Add(-24 * time.Hour),
		State:     "open",
		User:      &user{Login: "external", Type: "User"},
		Assignees: []user{},
		Labels:    []label{},
	}
}

func assignedTo(pr pullRequest, names ...string) pullRequest {
	for _, name := range names {
		pr.Assignees = append(pr.Assignees, user{Login: name})
	}
	return pr
}
