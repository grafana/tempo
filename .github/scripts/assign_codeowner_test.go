package main

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testNow = time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)

func firstChoice(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[0]
}

func testPull(number int) pullRequest {
	return pullRequest{
		Number:    number,
		CreatedAt: testNow.Add(-24 * time.Hour),
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

func TestParseCodeownersCatchAllDuplicatesAndExclusions(t *testing.T) {
	groups := parseCodeowners(`
# A comment is not a rule.
* @Alice @alice @grafana/tempo
/docs/ @grafana/docs-traces @Bob
/tools/ @Bob
`, "@grafana/docs-traces")
	require.Equal(t, []string{"alice"}, slices.Sorted(maps.Keys(groups.users)))
	require.Equal(t, []string{"grafana/tempo"}, slices.Sorted(maps.Keys(groups.teams)))
	require.Equal(t, []string{"bob"}, slices.Sorted(maps.Keys(groups.partial)))
	require.Equal(t, []string{"grafana/docs-traces"}, slices.Sorted(maps.Keys(groups.excluded)))
}

func TestLoadDeduplicatesAndFiltersBots(t *testing.T) {
	bot := assignedTo(testPull(4), "Bob")
	bot.User = &user{Login: "dependabot[bot]", Type: "Bot"}
	openPRs := []pullRequest{assignedTo(testPull(1), "Alice", "Alice"), bot}
	closedPRs := []pullRequest{assignedTo(testPull(4), "Alice", "Alice")}
	load, staleOwners := calculateLoad(openPRs, closedPRs, []string{"alice", "bob"}, testNow, 90)
	require.Equal(t, ownerLoad{open: 1, closed: 1}, load["alice"])
	require.Equal(t, ownerLoad{}, load["bob"])
	require.Empty(t, staleOwners)
}

func TestLoadExpiryBoundary(t *testing.T) {
	maxAge := 90 * 24 * time.Hour
	for _, test := range []struct {
		name      string
		elapsed   time.Duration
		wantOpen  int
		wantStale bool
	}{
		{name: "just below", elapsed: maxAge - time.Nanosecond, wantOpen: 1},
		{name: "exact boundary", elapsed: maxAge, wantOpen: 1},
		{name: "just above", elapsed: maxAge + time.Nanosecond, wantStale: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			pr := assignedTo(testPull(1), "Alice")
			pr.CreatedAt = testNow.Add(-test.elapsed)
			load, stale := calculateLoad([]pullRequest{pr}, nil, []string{"alice"}, testNow, 90)
			require.Equal(t, test.wantOpen, load["alice"].open)
			_, isStale := stale["alice"]
			require.Equal(t, test.wantStale, isStale)
		})
	}
}

func TestCandidatesFilterAndOrderByPRNumber(t *testing.T) {
	marked := testPull(2)
	marked.Labels = []label{{Name: defaultMarkerLabel}}
	bot := testPull(6)
	bot.User = &user{Login: "dependabot[bot]", Type: "Bot"}
	deletedAuthor := testPull(4)
	deletedAuthor.User = nil
	draft := testPull(3)
	draft.Draft = true
	ownerAuthored := testPull(7)
	ownerAuthored.User.Login = "Alice"
	assigned := assignedTo(testPull(5), "Bob")
	candidates := findCandidates([]pullRequest{ownerAuthored, assigned, marked, testPull(1), bot, deletedAuthor, draft}, []string{"alice", "bob"}, defaultMarkerLabel)
	require.Equal(t, []candidate{{number: 1}, {number: 2, marked: true}}, candidates)
}

func TestAssigneeTieBreakPrefersLessOpenLoadThenUsesExactTies(t *testing.T) {
	load := map[string]ownerLoad{
		"alice": {open: 1, closed: 1},
		"bob":   {open: 0, closed: 2},
		"carol": {open: 3},
	}
	chosen, ok := chooseAssignee(load, firstChoice)
	require.True(t, ok)
	require.Equal(t, "bob", chosen)
	load["bob"] = ownerLoad{open: 1, closed: 2}
	chosen, ok = chooseAssignee(load, firstChoice)
	require.True(t, ok)
	require.Equal(t, "alice", chosen)
}

func TestRandomChoiceUsesProductionIndexAndHandlesEmpty(t *testing.T) {
	calls := 0
	intN := func(limit int) int {
		calls++
		require.Equal(t, 3, limit)
		return 2
	}
	require.Equal(t, "carol", randomChoice([]string{"alice", "bob", "carol"}, intN))
	require.Equal(t, 1, calls)
	require.Empty(t, randomChoice(nil, func(int) int {
		t.Fatal("random index must not be called for an empty choice set")
		return 0
	}))
	require.Empty(t, firstChoice(nil))
}

func TestOnlyHumanUnassignAfterLatestBotAssignmentIsSkipped(t *testing.T) {
	assigned := timelineEvent{Event: "assigned", Actor: &user{Login: defaultBotActor}}
	humanUnassigned := timelineEvent{Event: "unassigned", Actor: &user{Login: "maintainer"}}
	automaticUnassigned := timelineEvent{Event: "unassigned"}
	require.True(t, deliberatelyUnassigned([]timelineEvent{assigned, humanUnassigned}, defaultBotActor))
	require.False(t, deliberatelyUnassigned([]timelineEvent{humanUnassigned, assigned}, defaultBotActor))
	require.False(t, deliberatelyUnassigned([]timelineEvent{assigned, automaticUnassigned}, defaultBotActor))
}

func TestSweepRetriesPaginatesAndAssignsOldestUpToCap(t *testing.T) {
	simulation, client := newSimulationClient(t)
	require.NoError(t, runTestSweep(client, testConfig()))
	require.Equal(t, 0, simulation.remainingRepoFailures())
	require.Equal(t, []user{{Login: "Alice"}}, simulation.pullSnapshot(10).Assignees)
	require.Empty(t, simulation.pullSnapshot(20).Assignees)
	require.Equal(t, []label{{Name: defaultMarkerLabel}}, simulation.pullSnapshot(10).Labels)
	require.True(t, simulation.hasLabel(defaultMarkerLabel))
}

func TestSweepSpreadsBatchAcrossTiedOwners(t *testing.T) {
	simulation, client := newSimulationClient(t)
	cfg := testConfig()
	cfg.maxAssignments = 2
	require.NoError(t, runTestSweep(client, cfg))
	require.Equal(t, []user{{Login: "Alice"}}, simulation.pullSnapshot(10).Assignees)
	require.Equal(t, []user{{Login: "Bob"}}, simulation.pullSnapshot(20).Assignees)
}

func TestSweepRollsBackAssignmentWhenMarkerFails(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFlag(&simulation.failMarkerApply)
	err := runTestSweep(client, testConfig())
	require.ErrorContains(t, err, "2 PR(s) could not be processed")
	for _, number := range []int{10, 20} {
		pr := simulation.pullSnapshot(number)
		require.Empty(t, pr.Assignees)
		require.Empty(t, pr.Labels)
	}
}

func TestSweepRollsBackOwnAssigneeWhenHumanRaces(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setConcurrentAssignee("Human")
	require.NoError(t, runTestSweep(client, testConfig()))
	for _, number := range []int{10, 20} {
		pr := simulation.pullSnapshot(number)
		require.Equal(t, []user{{Login: "Human"}}, pr.Assignees)
		require.Empty(t, pr.Labels)
	}
}

func TestSimulationRejectsUnknownPRAndInvalidJSON(t *testing.T) {
	simulation := newGitHubSimulation()
	for _, test := range []struct {
		method string
		path   string
		body   string
		status int
	}{
		{method: http.MethodPost, path: "/repos/grafana/tempo/issues/99/assignees", body: `{"assignees":["alice"]}`, status: http.StatusNotFound},
		{method: http.MethodPost, path: "/repos/grafana/tempo/issues/99/labels", body: `{"labels":["codeowner-sweep"]}`, status: http.StatusNotFound},
		{method: http.MethodPost, path: "/repos/grafana/tempo/issues/10/assignees", body: `{`, status: http.StatusBadRequest},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		simulation.ServeHTTP(recorder, request)
		require.Equal(t, test.status, recorder.Code)
	}
}

func TestFailureReportCreatesThenReusesIssue(t *testing.T) {
	simulation, client := newSimulationClient(t)
	const runURL = "https://github.com/grafana/tempo/actions/runs/123"
	require.NoError(t, reportFailure(context.Background(), client, simulation.repo, runURL))
	created := simulation.issueSnapshot()
	require.Len(t, created, 1)
	require.Contains(t, created[0].Body, runURL)

	simulation.setFailureIssue(42)
	require.NoError(t, reportFailure(context.Background(), client, simulation.repo, runURL))
	require.Len(t, simulation.issueSnapshot(), 1)
	require.Equal(t, []issueComment{{Number: 42, Body: created[0].Body}}, simulation.commentSnapshot())
}

func TestFailureReportSearchErrorDoesNotCreateDuplicate(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFlag(&simulation.failIssueSearch)
	err := reportFailure(context.Background(), client, simulation.repo, "https://example.invalid/run")
	require.ErrorContains(t, err, "find existing sweep failure issue")
	require.Empty(t, simulation.issueSnapshot())
	require.Empty(t, simulation.commentSnapshot())
}
