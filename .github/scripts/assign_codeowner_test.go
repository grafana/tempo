package main

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestLoadFutureCreatedAtCountsAsOpen(t *testing.T) {
	pr := assignedTo(testPull(1), "Alice")
	pr.CreatedAt = testNow.Add(24 * time.Hour)
	load, stale := calculateLoad([]pullRequest{pr}, nil, []string{"alice"}, testNow, 90)
	require.Equal(t, ownerLoad{open: 1}, load["alice"])
	require.Empty(t, stale)
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

func TestCandidateMarkingIgnoresLabelCase(t *testing.T) {
	pr := testPull(1)
	pr.Labels = []label{{Name: "Codeowner-Sweep"}}
	candidates := findCandidates([]pullRequest{pr}, []string{"alice"}, defaultMarkerLabel)
	require.Equal(t, []candidate{{number: 1, marked: true}}, candidates)
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
	_, ok = chooseAssignee(map[string]ownerLoad{}, firstChoice)
	require.False(t, ok)
}

func TestDecideAssignmentsSpreadsAcrossLeastLoadedOwners(t *testing.T) {
	load := map[string]ownerLoad{
		"alice": {open: 1},
		"bob":   {},
	}
	candidates := []candidate{{number: 10}, {number: 20}, {number: 30}}
	assignments := decideAssignments(load, candidates, 2, firstChoice)
	require.Equal(t, []assignment{{number: 10, owner: "bob"}, {number: 20, owner: "alice"}}, assignments)
}

func TestDecideAssignmentsTiesUseChooserAndHonorCap(t *testing.T) {
	load := map[string]ownerLoad{"alice": {}, "bob": {}}
	candidates := []candidate{{number: 10}, {number: 20}, {number: 30}, {number: 40}, {number: 50}}
	assignments := decideAssignments(load, candidates, 4, firstChoice)
	require.Equal(t, []assignment{
		{number: 10, owner: "alice"},
		{number: 20, owner: "bob"},
		{number: 30, owner: "alice"},
		{number: 40, owner: "bob"},
	}, assignments)
}

func TestOnlyHumanUnassignAfterLatestBotAssignmentIsSkipped(t *testing.T) {
	assigned := timelineEvent{Event: "assigned", Actor: &user{Login: defaultBotActor}}
	humanUnassigned := timelineEvent{Event: "unassigned", Actor: &user{Login: "maintainer"}}
	automaticUnassigned := timelineEvent{Event: "unassigned"}
	require.True(t, deliberatelyUnassigned([]timelineEvent{assigned, humanUnassigned}, defaultBotActor))
	require.False(t, deliberatelyUnassigned([]timelineEvent{humanUnassigned, assigned}, defaultBotActor))
	require.False(t, deliberatelyUnassigned([]timelineEvent{assigned, automaticUnassigned}, defaultBotActor))
}

func TestSweepPaginatesAndAssignsUpToCap(t *testing.T) {
	simulation, client := newSimulationClient(t)
	require.NoError(t, runTestSweep(client, testConfig()))
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

func TestSweepCountsClosedLoadInChoice(t *testing.T) {
	simulation, client := newSimulationClient(t)
	// Alice carries one recently closed PR; Bob carries none, so Bob has the
	// lower total load and must be chosen for the open PR.
	simulation.setClosedPulls([]pullRequest{assignedTo(testPull(99), "Alice")})
	require.NoError(t, runTestSweep(client, testConfig()))
	require.Equal(t, []user{{Login: "Bob"}}, simulation.pullSnapshot(10).Assignees)
}

func TestSweepSkipsMarkedCandidateUnassignedByAHuman(t *testing.T) {
	simulation, client := newSimulationClient(t)
	cfg := testConfig()
	cfg.maxAssignments = 3
	marked := testPull(30)
	marked.Labels = []label{{Name: defaultMarkerLabel}}
	simulation.pulls[30] = &marked
	simulation.setTimeline(30, []timelineEvent{
		{Event: "assigned", Actor: &user{Login: defaultBotActor}},
		{Event: "unassigned", Actor: &user{Login: "maintainer"}},
	})
	require.NoError(t, runTestSweep(client, cfg))
	require.Equal(t, []user{{Login: "Alice"}}, simulation.pullSnapshot(10).Assignees)
	require.Equal(t, []user{{Login: "Bob"}}, simulation.pullSnapshot(20).Assignees)
	skipped := simulation.pullSnapshot(30)
	require.Empty(t, skipped.Assignees)
	require.Equal(t, []label{{Name: defaultMarkerLabel}}, skipped.Labels)
}

func TestSweepLeavesClosedCandidateAlone(t *testing.T) {
	simulation, client := newSimulationClient(t)
	closed := testPull(30)
	closed.State = "closed"
	simulation.pulls[30] = &closed
	cfg := testConfig()
	cfg.maxAssignments = 3
	require.NoError(t, runTestSweep(client, cfg))
	// The candidate snapshot listed PR #30 as open; by the time it was
	// re-read before assignment it was closed, so it must not be assigned.
	require.Empty(t, simulation.pullSnapshot(30).Assignees)
	require.Equal(t, []user{{Login: "Alice"}}, simulation.pullSnapshot(10).Assignees)
}

func TestSweepFailsFastWhenMarkerLabelCannotBeEnsured(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFlag(&simulation.failEnsureLabel)
	err := runTestSweep(client, testConfig())
	require.ErrorContains(t, err, "ensure marker label")
	require.Empty(t, simulation.pullSnapshot(10).Assignees)
	require.Empty(t, simulation.pullSnapshot(20).Assignees)
}

func TestSweepAbortsBatchWhenMarkerCannotBeApplied(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFlag(&simulation.failMarkerApply)
	err := runTestSweep(client, testConfig())
	require.ErrorContains(t, err, "marker label")
	for _, number := range []int{10, 20} {
		pr := simulation.pullSnapshot(number)
		require.Empty(t, pr.Assignees)
		require.Empty(t, pr.Labels)
	}
	// Only the first candidate was attempted: the batch aborted instead of
	// assigning and rolling back every PR in one run.
	require.Equal(t, 1, simulation.assignAttemptCount())
}

func TestSweepNeverRetriesAssignmentPost(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFlag(&simulation.failAssignPost)
	err := runTestSweep(client, testConfig())
	require.ErrorContains(t, err, "could not be processed")
	// POST /assignees is not idempotent: a 500 must run exactly once rather
	// than being replayed by the retry loop.
	require.Equal(t, 1, simulation.assignAttemptCount())
}

func TestSweepRollsBackOwnAssigneeWhenHumanRaces(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setConcurrentAssignee("Human")
	require.NoError(t, runTestSweep(client, testConfig()))
	// The raced assignment consumed the per-run cap; PR #20 waits for the
	// next run while PR #10 keeps only the human's assignee.
	require.Equal(t, []user{{Login: "Human"}}, simulation.pullSnapshot(10).Assignees)
	require.Empty(t, simulation.pullSnapshot(20).Assignees)
	require.Empty(t, simulation.pullSnapshot(10).Labels)
}

func TestAssignAndMarkConcurrentRollbackFailureSurfacesError(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setConcurrentAssignee("Human")
	simulation.setFlag(&simulation.failRollbackDelete)
	_, err := assignAndMark(context.Background(), client, testConfig(), 10, "alice")
	require.ErrorContains(t, err, "concurrent assignment detected; rollback failed")
	require.Equal(t, []user{{Login: "Human"}, {Login: "Alice"}}, simulation.pullSnapshot(10).Assignees)
}

func TestAssignAndMarkMarkerRollbackFailureJoinsErrors(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFlag(&simulation.failMarkerApply)
	simulation.setFlag(&simulation.failRollbackDelete)
	_, err := assignAndMark(context.Background(), client, testConfig(), 10, "alice")
	require.ErrorContains(t, err, "apply marker label")
	require.ErrorContains(t, err, "rollback assignee")
	require.Equal(t, []user{{Login: "Alice"}}, simulation.pullSnapshot(10).Assignees)
}

func TestFailureReportCommentFailureSurfaces(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFailureIssue(42)
	simulation.setFlag(&simulation.failIssueComments)
	err := reportFailure(context.Background(), client, simulation.repo, "https://example.invalid/run")
	require.ErrorContains(t, err, "comment on sweep failure issue #42")
	require.Empty(t, simulation.issueSnapshot())
}

func TestFailureReportCreateFailureSurfaces(t *testing.T) {
	simulation, client := newSimulationClient(t)
	simulation.setFlag(&simulation.failIssueCreate)
	err := reportFailure(context.Background(), client, simulation.repo, "https://example.invalid/run")
	require.ErrorContains(t, err, "create sweep failure issue")
	require.Empty(t, simulation.issueSnapshot())
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
		{method: http.MethodPost, path: "/repos/grafana/tempo/issues/99/labels", body: `["codeowner-sweep"]`, status: http.StatusNotFound},
		{method: http.MethodPost, path: "/repos/grafana/tempo/issues/10/assignees", body: `{`, status: http.StatusBadRequest},
		{method: http.MethodPost, path: "/repos/grafana/tempo/issues/10/assignees", body: `{"assignees":["unknown-login"]}`, status: http.StatusUnprocessableEntity},
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
