package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

func TestConfigFromEnvValidatesRequiredValues(t *testing.T) {
	for _, test := range []struct {
		name      string
		repo      string
		max       string
		age       string
		unsetRepo bool
		wantErr   string
	}{
		{name: "missing repo", unsetRepo: true, max: "25", age: "90", wantErr: "missing required environment variable: REPO"},
		{name: "invalid repo", repo: "tempo", max: "25", age: "90", wantErr: "owner/repository"},
		{name: "invalid assignment integer", repo: "grafana/tempo", max: "many", age: "90", wantErr: "parse MAX_ASSIGNMENTS"},
		{name: "zero assignment cap", repo: "grafana/tempo", max: "0", age: "90", wantErr: "MAX_ASSIGNMENTS must be greater than zero"},
		{name: "zero open age", repo: "grafana/tempo", max: "25", age: "0", wantErr: "OPEN_LOAD_MAX_AGE_DAYS must be greater than zero"},
		{name: "negative assignment cap", repo: "grafana/tempo", max: "-1", age: "90", wantErr: "MAX_ASSIGNMENTS must be greater than zero"},
		{name: "empty marker label", repo: "grafana/tempo", max: "25", age: "90", wantErr: "MARKER_LABEL must not be empty"},
		{name: "empty bot actor", repo: "grafana/tempo", max: "25", age: "90", wantErr: "BOT_ACTOR must not be empty"},
		{name: "age duration overflow", repo: "grafana/tempo", max: "25", age: "106752", wantErr: "must not exceed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.unsetRepo {
				unsetEnvironment(t, "REPO")
			} else {
				t.Setenv("REPO", test.repo)
			}
			t.Setenv("MAX_ASSIGNMENTS", test.max)
			t.Setenv("OPEN_LOAD_MAX_AGE_DAYS", test.age)
			t.Setenv("MARKER_LABEL", defaultMarkerLabel)
			t.Setenv("BOT_ACTOR", defaultBotActor)
			switch test.name {
			case "empty marker label":
				t.Setenv("MARKER_LABEL", "")
			case "empty bot actor":
				t.Setenv("BOT_ACTOR", "")
			}
			_, err := configFromEnv()
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestConfigDefaultsAndWorkflowContract(t *testing.T) {
	t.Setenv("REPO", "grafana/tempo")
	t.Setenv("MAX_ASSIGNMENTS", "25")
	t.Setenv("OPEN_LOAD_MAX_AGE_DAYS", "90")
	t.Setenv("ASSIGNEE_POOL", "Alice,Bob")
	t.Setenv("EXCLUDED_OWNERS", "grafana/docs-traces")
	t.Setenv("MARKER_LABEL", defaultMarkerLabel)
	t.Setenv("BOT_ACTOR", defaultBotActor)
	cfg, err := configFromEnv()
	require.NoError(t, err)
	require.Equal(t, defaultConfig("grafana/tempo").maxAssignments, cfg.maxAssignments)
	require.Equal(t, defaultConfig("grafana/tempo").openLoadMaxAgeDays, cfg.openLoadMaxAgeDays)

	data, err := os.ReadFile("../workflows/assign-codeowner.yml")
	require.NoError(t, err)
	workflow := string(data)
	for _, expected := range []string{
		"GH_TOKEN: ${{ github.token }}",
		"REPO: ${{ github.repository }}",
		"MAX_ASSIGNMENTS: '25'",
		"OPEN_LOAD_MAX_AGE_DAYS: '90'",
		"MARKER_LABEL: " + defaultMarkerLabel,
		"BOT_ACTOR: '" + defaultBotActor + "'",
		"ASSIGNEE_POOL: ${{ vars.CODEOWNER_ASSIGNEES }}",
		"EXCLUDED_OWNERS: 'grafana/docs-traces'",
		"RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}",
		"working-directory: .github/scripts",
		"run: go run .",
		"run: go run . report-failure",
		"cron: '0 7 * * *'",
		"permissions: {}",
	} {
		require.Contains(t, workflow, expected, "workflow contract string missing: %q", expected)
	}
}

func TestConfigFromEnvFallsBackToHardcodedPool(t *testing.T) {
	t.Setenv("REPO", "grafana/tempo")
	t.Setenv("MAX_ASSIGNMENTS", "25")
	t.Setenv("OPEN_LOAD_MAX_AGE_DAYS", "90")
	t.Setenv("MARKER_LABEL", defaultMarkerLabel)
	t.Setenv("BOT_ACTOR", defaultBotActor)
	for _, test := range []struct {
		name string
		env  string
		want string
	}{
		{name: "env unset falls back", env: "", want: defaultAssigneePool},
		{name: "non-empty env overrides", env: "Alice,Bob", want: "Alice,Bob"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("ASSIGNEE_POOL", test.env)
			cfg, err := configFromEnv()
			require.NoError(t, err)
			require.Equal(t, test.want, cfg.assigneePool)
		})
	}
}

func TestRunReportFailureBypassesSweepConfigValidation(t *testing.T) {
	t.Setenv("GH_TOKEN", "token")
	t.Setenv("REPO", "grafana/tempo")
	t.Setenv("MAX_ASSIGNMENTS", "abc")
	t.Setenv("OPEN_LOAD_MAX_AGE_DAYS", "90")
	unsetEnvironment(t, "RUN_URL")
	err := run(context.Background(), "report-failure")
	require.ErrorContains(t, err, "RUN_URL")
	require.NotContains(t, err.Error(), "MAX_ASSIGNMENTS")
	err = run(context.Background(), "sweep")
	require.ErrorContains(t, err, "parse MAX_ASSIGNMENTS")
}

func TestResolvePoolFiltersPoolToAssignableAndExcluded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/grafana/tempo/assignees", r.URL.Path)
		fmt.Fprint(w, `[{"login":"Alice"},{"login":"Bob"},{"login":"Carol"}]`)
	}))
	defer server.Close()
	client := testGitHubClient(t, server.URL)
	cfg := defaultConfig("grafana/tempo")
	cfg.assigneePool = "Alice, Bob, Unassignable"
	cfg.excludedOwners = "@Bob"
	eligible, err := resolvePool(context.Background(), client, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"alice"}, eligible)
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
