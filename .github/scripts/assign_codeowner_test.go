package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

// applyStub records the requests an applyAssignment test server received.
type applyStub struct {
	mu       sync.Mutex
	requests []string
}

func (s *applyStub) record(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, r.Method+" "+r.URL.Path)
}

func (s *applyStub) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func TestApplyAssignmentAssignsAndMarks(t *testing.T) {
	stub := &applyStub{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.record(r)
		switch {
		case r.Method == http.MethodGet:
			fmt.Fprint(w, `{"number":10,"state":"open"}`)
		case strings.HasSuffix(r.URL.Path, "/assignees"):
			fmt.Fprint(w, `{"number":10,"assignees":[{"login":"alice"}]}`)
		case strings.HasSuffix(r.URL.Path, "/labels"):
			fmt.Fprint(w, `[]`)
		default:
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	outcome, err := applyAssignment(context.Background(), testGitHubClient(t, server.URL), testConfig(), 10, "alice")
	require.NoError(t, err)
	require.Equal(t, applySuccess, outcome)
	require.Equal(t, []string{
		"GET /repos/grafana/tempo/issues/10",
		"POST /repos/grafana/tempo/issues/10/assignees",
		"POST /repos/grafana/tempo/issues/10/labels",
	}, stub.snapshot())
}

func TestApplyAssignmentSkipsAssignedOrClosedCandidates(t *testing.T) {
	for _, test := range []struct {
		name  string
		issue string
	}{
		{name: "already assigned", issue: `{"number":10,"state":"open","assignees":[{"login":"alice"}]}`},
		{name: "closed", issue: `{"number":10,"state":"closed"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &applyStub{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				stub.record(r)
				fmt.Fprint(w, test.issue)
			}))
			defer server.Close()
			outcome, err := applyAssignment(context.Background(), testGitHubClient(t, server.URL), testConfig(), 10, "alice")
			require.NoError(t, err)
			require.Equal(t, applySkipped, outcome)
			require.Equal(t, []string{"GET /repos/grafana/tempo/issues/10"}, stub.snapshot())
		})
	}
}

func TestApplyAssignmentRollsBackAndAbortsWhenMarkerFails(t *testing.T) {
	stub := &applyStub{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.record(r)
		switch {
		case r.Method == http.MethodGet:
			fmt.Fprint(w, `{"number":10,"state":"open"}`)
		case strings.HasSuffix(r.URL.Path, "/assignees") && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"number":10,"assignees":[{"login":"alice"}]}`)
		case strings.HasSuffix(r.URL.Path, "/labels"):
			http.Error(w, "label write failed", http.StatusInternalServerError)
		case strings.HasSuffix(r.URL.Path, "/assignees") && r.Method == http.MethodDelete:
			fmt.Fprint(w, `{"number":10,"assignees":[]}`)
		default:
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	outcome, err := applyAssignment(context.Background(), testGitHubClient(t, server.URL), testConfig(), 10, "alice")
	require.ErrorContains(t, err, "apply marker label")
	require.Equal(t, applyAbort, outcome)
	require.Equal(t, []string{
		"GET /repos/grafana/tempo/issues/10",
		"POST /repos/grafana/tempo/issues/10/assignees",
		"POST /repos/grafana/tempo/issues/10/labels",
		"DELETE /repos/grafana/tempo/issues/10/assignees",
	}, stub.snapshot())
}

func TestApplyAssignmentSurfacesRollbackFailure(t *testing.T) {
	zeroRetryDelay(t)
	stub := &applyStub{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.record(r)
		switch {
		case r.Method == http.MethodGet:
			fmt.Fprint(w, `{"number":10,"state":"open"}`)
		case strings.HasSuffix(r.URL.Path, "/assignees") && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"number":10,"assignees":[{"login":"alice"}]}`)
		case strings.HasSuffix(r.URL.Path, "/labels"):
			http.Error(w, "label write failed", http.StatusInternalServerError)
		default:
			http.Error(w, "rollback unavailable", http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	outcome, err := applyAssignment(context.Background(), testGitHubClient(t, server.URL), testConfig(), 10, "alice")
	require.ErrorContains(t, err, "apply marker label")
	require.ErrorContains(t, err, "rollback assignee")
	require.Equal(t, applyAbort, outcome)
}

func TestApplyAssignmentRollsBackWhenAnotherAssigneeRaces(t *testing.T) {
	stub := &applyStub{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.record(r)
		switch {
		case r.Method == http.MethodGet:
			fmt.Fprint(w, `{"number":10,"state":"open"}`)
		case strings.HasSuffix(r.URL.Path, "/assignees") && r.Method == http.MethodPost:
			// The response shows a second, racing assignee.
			fmt.Fprint(w, `{"number":10,"assignees":[{"login":"alice"},{"login":"human"}]}`)
		case strings.HasSuffix(r.URL.Path, "/assignees") && r.Method == http.MethodDelete:
			fmt.Fprint(w, `{"number":10,"assignees":[]}`)
		case strings.HasSuffix(r.URL.Path, "/labels"):
			fmt.Fprint(w, `[]`)
		default:
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	outcome, err := applyAssignment(context.Background(), testGitHubClient(t, server.URL), testConfig(), 10, "alice")
	require.NoError(t, err)
	require.Equal(t, applySkipped, outcome)
	require.Equal(t, []string{
		"GET /repos/grafana/tempo/issues/10",
		"POST /repos/grafana/tempo/issues/10/assignees",
		"DELETE /repos/grafana/tempo/issues/10/assignees",
	}, stub.snapshot())
}
