package main

import (
	"context"
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

func TestSettingsAndWorkflowContract(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/assign-codeowner.yml")
	require.NoError(t, err)
	workflow := string(data)
	for _, expected := range []string{
		"GH_TOKEN: ${{ github.token }}",
		"working-directory: tools",
		"run: go run ./assign-codeowner",
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

func TestRunRequiresToken(t *testing.T) {
	unsetEnvironment(t, "GH_TOKEN")
	err := run(context.Background())
	require.ErrorContains(t, err, "missing required environment variable: GH_TOKEN")
}

func TestEachPageBoundedStopsAtLimit(t *testing.T) {
	calls := 0
	items, err := eachPageBounded(context.Background(), func(opts *github.ListOptions) ([]int, *github.Response, error) {
		calls++
		return []int{1, 2, 3}, &github.Response{NextPage: opts.Page + 1}, nil
	}, 5)
	require.NoError(t, err)
	// Stops as soon as the cap is reached and truncates to it: page 1 (3 items)
	// + page 2 would reach 6, so the result is cut to 5 without a third request.
	require.Equal(t, []int{1, 2, 3, 1, 2}, items)
	require.Equal(t, 2, calls)
}
