package main

import (
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
	bot := testPull(6)
	bot.User = &user{Login: "dependabot[bot]", Type: "Bot"}
	deletedAuthor := testPull(4)
	deletedAuthor.User = nil
	draft := testPull(3)
	draft.Draft = true
	ownerAuthored := testPull(7)
	ownerAuthored.User.Login = "Alice"
	assigned := assignedTo(testPull(5), "Bob")
	candidates := findCandidates([]pullRequest{ownerAuthored, assigned, testPull(1), bot, deletedAuthor, draft}, []string{"alice", "bob"})
	require.Equal(t, []int{1}, candidates)
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

// TestCompareLoadRecentlyClosedIsOnlyATiebreaker pins that open load is the
// primary ordering signal and recent closures only break exact open-load ties.
// Under a sum-first ordering this regression case picks the wrong owner, so it
// guards against closed load ever outranking an emptier open queue.
func TestCompareLoadRecentlyClosedIsOnlyATiebreaker(t *testing.T) {
	// A has zero open PRs but a busy 7-day closure history; B has one open PR
	// and no closures. A is the least-loaded owner and must win.
	require.Less(t, compareLoad(ownerLoad{open: 0, closed: 9}, ownerLoad{open: 1, closed: 0}), 0)
	// Equal open load: fewer recent closures wins the tie.
	require.Less(t, compareLoad(ownerLoad{open: 1, closed: 0}, ownerLoad{open: 1, closed: 5}), 0)
	// Exact equality is a tie.
	require.Equal(t, 0, compareLoad(ownerLoad{open: 1, closed: 2}, ownerLoad{open: 1, closed: 2}))
}

func TestDecideAssignmentsSpreadsAcrossLeastLoadedOwners(t *testing.T) {
	load := map[string]ownerLoad{
		"alice": {open: 1},
		"bob":   {},
	}
	candidates := []int{10, 20, 30}
	assignments := decideAssignments(load, candidates, 2, firstChoice)
	require.Equal(t, []assignment{{number: 10, owner: "bob"}, {number: 20, owner: "alice"}}, assignments)
}

func TestDecideAssignmentsTiesUseChooserAndHonorCap(t *testing.T) {
	load := map[string]ownerLoad{"alice": {}, "bob": {}}
	candidates := []int{10, 20, 30, 40, 50}
	assignments := decideAssignments(load, candidates, 4, firstChoice)
	require.Equal(t, []assignment{
		{number: 10, owner: "alice"},
		{number: 20, owner: "bob"},
		{number: 30, owner: "alice"},
		{number: 40, owner: "bob"},
	}, assignments)
}
