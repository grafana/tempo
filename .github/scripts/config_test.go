package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/stretchr/testify/require"
)

func unsetEnvironment(t *testing.T, name string) {
	t.Helper()
	value, present := os.LookupEnv(name)
	t.Cleanup(func() {
		if present {
			_ = os.Setenv(name, value)
		} else {
			_ = os.Unsetenv(name)
		}
	})
	require.NoError(t, os.Unsetenv(name))
}

func TestConfigDefaultsAndWorkflowContract(t *testing.T) {
	cfg := defaultConfig()
	require.Equal(t, defaultRepo, cfg.repo)
	require.Equal(t, defaultAssigneePool, cfg.assigneePool)
	require.Equal(t, defaultMaxAssignments, cfg.maxAssignments)
	require.Equal(t, defaultOpenLoadMaxAgeDays, cfg.openLoadMaxAgeDays)
	require.Equal(t, defaultMarkerLabel, cfg.markerLabel)
	require.Equal(t, defaultBotActor, cfg.botActor)

	data, err := os.ReadFile("../workflows/assign-codeowner.yml")
	require.NoError(t, err)
	workflow := string(data)
	for _, expected := range []string{
		"GH_TOKEN: ${{ github.token }}",
		"RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}",
		"working-directory: .github/scripts",
		"run: go run .",
		"run: go run . report-failure",
		"cron: '0 7 * * *'",
		"permissions: {}",
	} {
		require.Contains(t, workflow, expected, "workflow contract string missing: %q", expected)
	}
	// Settings live only in Go now; a workflow env block would silently drift.
	for _, gone := range []string{"ASSIGNEE_POOL", "MAX_ASSIGNMENTS", "OPEN_LOAD_MAX_AGE_DAYS", "MARKER_LABEL", "BOT_ACTOR", "EXCLUDED_OWNERS", "REPO:"} {
		require.NotContains(t, workflow, gone+" ", "stale workflow env: %q", gone)
	}
}

func TestRunRequiresTokenAndReportFailureRequiresRunURL(t *testing.T) {
	unsetEnvironment(t, "GH_TOKEN")
	err := run(context.Background(), "sweep")
	require.ErrorContains(t, err, "missing required environment variable: GH_TOKEN")

	t.Setenv("GH_TOKEN", "token")
	unsetEnvironment(t, "RUN_URL")
	err = run(context.Background(), "report-failure")
	require.ErrorContains(t, err, "missing required environment variable: RUN_URL")
}

func TestResolvePoolFiltersPoolToAssignableAndExcluded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/grafana/tempo/assignees", r.URL.Path)
		fmt.Fprint(w, `[{"login":"Alice"},{"login":"Bob"},{"login":"Carol"}]`)
	}))
	defer server.Close()
	client := testGitHubClient(t, server.URL)
	cfg := defaultConfig()
	cfg.assigneePool = "Alice, Bob, Unassignable"
	cfg.excludedOwners = "@Bob"
	eligible, err := resolvePool(context.Background(), client, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"alice"}, eligible)
}

func TestEachPageBoundedStopsAtLimit(t *testing.T) {
	calls := 0
	items, err := eachPageBounded(context.Background(), func(opts *github.ListOptions) ([]int, *github.Response, error) {
		calls++
		return []int{1, 2, 3}, &github.Response{NextPage: opts.Page + 1}, nil
	}, 5)
	require.NoError(t, err)
	// Stops as soon as the cap is reached: page 1 (3 items) + page 2 fills past
	// the 5-item limit without requesting a third page.
	require.Equal(t, []int{1, 2, 3, 1, 2, 3}, items)
	require.Equal(t, 2, calls)
}

func TestEnsureMarkerLabelPreservesNon404Error(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodPatch, r.Method)
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()
	client := testGitHubClient(t, server.URL)
	err := ensureMarkerLabel(context.Background(), client, testConfig())
	require.ErrorContains(t, err, "update marker label")
	require.Equal(t, 1, calls)
}

func TestEnsureMarkerLabelCreatesMissingLabel(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		switch r.Method {
		case http.MethodPatch:
			http.Error(w, "missing", http.StatusNotFound)
		case http.MethodPost:
			fmt.Fprint(w, `{"name":"codeowner-sweep"}`)
		default:
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client := testGitHubClient(t, server.URL)
	require.NoError(t, ensureMarkerLabel(context.Background(), client, testConfig()))
	require.Equal(t, []string{http.MethodPatch, http.MethodPost}, methods)
}

func TestQueryWithPanicsOnOddArguments(t *testing.T) {
	require.Panics(t, func() { queryWith("only-key") })
	values := queryWith("q", "value", "per_page", "1")
	require.Equal(t, "value", values.Get("q"))
	require.Equal(t, "1", values.Get("per_page"))
}
