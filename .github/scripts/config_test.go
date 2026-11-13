package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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
		"MAX_ASSIGNMENTS: '25'",
		"OPEN_LOAD_MAX_AGE_DAYS: '90'",
		"MARKER_LABEL: " + defaultMarkerLabel,
		"BOT_ACTOR: '" + defaultBotActor + "'",
		"EXCLUDED_OWNERS: 'grafana/docs-traces'",
		"run: go run ./.github/scripts",
		"run: go run ./.github/scripts report-failure",
	} {
		require.Contains(t, workflow, expected)
	}
}

func TestReadCodeownersErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		content string
		wantErr string
	}{
		{name: "missing everywhere", status: http.StatusNotFound, wantErr: "no CODEOWNERS file found"},
		{name: "server failure preserved", status: http.StatusInternalServerError, wantErr: "HTTP 500"},
		{name: "invalid base64", status: http.StatusOK, content: "not-base64", wantErr: "decode CODEOWNERS"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repos/grafana/tempo" {
					fmt.Fprint(w, `{"default_branch":"main"}`)
					return
				}
				if test.status == http.StatusOK {
					fmt.Fprintf(w, `{"content":%q}`, test.content)
					return
				}
				http.Error(w, "failure", test.status)
			}))
			defer server.Close()
			client := newGitHubClient("test")
			client.baseURL = server.URL
			client.retryDelay = func(int) time.Duration { return 0 }
			_, err := readCodeowners(context.Background(), client, "grafana/tempo")
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestReadCodeownersUsesFallbackPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/grafana/tempo":
			fmt.Fprint(w, `{"default_branch":"main"}`)
		case "/repos/grafana/tempo/contents/CODEOWNERS":
			http.Error(w, "missing", http.StatusNotFound)
		case "/repos/grafana/tempo/contents/.github/CODEOWNERS":
			encoded := base64.StdEncoding.EncodeToString([]byte("* @Alice\n"))
			fmt.Fprintf(w, `{"content":%q}`, encoded)
		default:
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newGitHubClient("test")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	text, err := readCodeowners(context.Background(), client, "grafana/tempo")
	require.NoError(t, err)
	require.Equal(t, "* @Alice\n", text)
}

func TestResolvePoolTeamFailureAndInvalidTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/grafana/tempo/assignees":
			fmt.Fprint(w, `[{"login":"Alice"},{"login":"Bob"}]`)
		case "/orgs/grafana/teams/tempo/members":
			http.Error(w, "unavailable", http.StatusInternalServerError)
		default:
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newGitHubClient("test")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	cfg := defaultConfig("grafana/tempo")
	groups := parseCodeowners("* @Alice @grafana/tempo", "")
	eligible, err := resolvePool(context.Background(), client, cfg, groups)
	require.NoError(t, err)
	require.Equal(t, []string{"alice"}, eligible)
	groups.teams["invalid"] = struct{}{}
	_, err = resolvePool(context.Background(), client, cfg, groups)
	require.ErrorContains(t, err, "expected org/team")
}

func TestEnsureMarkerLabelPreservesNon404Error(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodPatch, r.Method)
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()
	client := newGitHubClient("test")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	err := ensureMarkerLabel(context.Background(), client, testConfig())
	require.ErrorContains(t, err, "update marker label")
	require.Equal(t, 1, calls)
}

func TestEnsureMarkerLabelEscapesName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/grafana/tempo/labels/codeowner%2Fsweep", r.URL.EscapedPath())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := newGitHubClient("test")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	cfg := testConfig()
	cfg.markerLabel = "codeowner/sweep"
	require.NoError(t, ensureMarkerLabel(context.Background(), client, cfg))
}

func TestQueryBuilders(t *testing.T) {
	require.Equal(t, "100", pageQuery().Get("per_page"))
	values := queryWith("q", "value", "per_page", "1")
	require.Equal(t, "value", values.Get("q"))
	require.Equal(t, "1", values.Get("per_page"))
	require.Panics(t, func() { queryWith("odd") })
	require.True(t, strings.HasPrefix(repositoryEndpoint("grafana/tempo", "/pulls"), "repos/grafana/tempo/"))
	require.Equal(t, "repos/grafana/tempo/issues/10/labels", issueEndpoint("grafana/tempo", 10, "/labels"))
}
