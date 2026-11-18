package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"

	// math/rand/v2 is fine for tie-breaking between equal-load owners; the
	// output is never used for security.
	"math/rand/v2" // nosemgrep: math-random-used
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v93/github"
)

type user struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

type pullRequest struct {
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"created_at"`
	User      *user     `json:"user"`
	Assignees []user    `json:"assignees"`
	Draft     bool      `json:"draft"`
}

type ownerLoad struct {
	open   int
	closed int
}

type prOwner struct {
	number int
	owner  string
}

type loadState uint8

const (
	openLoad loadState = iota
	closedLoad
)

func calculateLoad(openPRs, closedPRs []pullRequest, eligible []string, now time.Time, maxOpenAgeDays int) (map[string]ownerLoad, map[string]struct{}) {
	load := make(map[string]ownerLoad, len(eligible))
	for _, owner := range eligible {
		load[owner] = ownerLoad{}
	}
	staleOwners := make(map[string]struct{})
	maxOpenAge := time.Duration(maxOpenAgeDays) * 24 * time.Hour
	for _, state := range []struct {
		kind  loadState
		pulls []pullRequest
	}{{openLoad, openPRs}, {closedLoad, closedPRs}} {
		seen := make(map[prOwner]struct{}, len(state.pulls))
		for _, pr := range state.pulls {
			if pr.User != nil && pr.User.Type == "Bot" {
				continue
			}
			for _, assignee := range pr.Assignees {
				owner := strings.ToLower(assignee.Login)
				key := prOwner{number: pr.Number, owner: owner}
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				if state.kind == openLoad {
					elapsed := now.Sub(pr.CreatedAt)
					// A future created_at (clock skew) must not wrap negative
					// and silently mark the owner stale; treat it as fresh
					// and let it count as load.
					if elapsed < 0 {
						elapsed = 0
					}
					if elapsed > maxOpenAge {
						staleOwners[owner] = struct{}{}
						continue
					}
				}
				counts, ok := load[owner]
				if !ok {
					continue
				}
				if state.kind == openLoad {
					counts.open++
				} else {
					counts.closed++
				}
				load[owner] = counts
			}
		}
	}
	return load, staleOwners
}

// candidateEligible centralizes candidate filtering and nil-author handling.
func candidateEligible(pr pullRequest, owners map[string]struct{}) bool {
	if len(pr.Assignees) > 0 || pr.Draft || pr.User == nil {
		return false
	}
	if pr.User.Type == "Bot" {
		return false
	}
	_, isOwner := owners[strings.ToLower(pr.User.Login)]
	return !isOwner
}

func findCandidates(pulls []pullRequest, eligible []string) []int {
	owners := make(map[string]struct{}, len(eligible))
	for _, owner := range eligible {
		owners[owner] = struct{}{}
	}
	result := make([]int, 0, len(pulls))
	for _, pr := range pulls {
		if candidateEligible(pr, owners) {
			result = append(result, pr.Number)
		}
	}
	slices.Sort(result)
	return result
}

// compareLoad orders owners by open load (a PR count is the primary signal of
// how busy an owner is), breaking ties by recently-closed load so an owner who
// just shed a batch stays at the back of the queue. The recently-closed count
// must never outweigh the open count.
func compareLoad(a, b ownerLoad) int {
	return cmp.Or(
		cmp.Compare(a.open, b.open),
		cmp.Compare(a.closed, b.closed),
	)
}

func chooseAssignee(load map[string]ownerLoad, choose func([]string) string) (string, bool) {
	var best ownerLoad
	choices := make([]string, 0, len(load))
	for _, owner := range slices.Sorted(maps.Keys(load)) {
		counts := load[owner]
		switch comparison := compareLoad(counts, best); {
		case len(choices) == 0 || comparison < 0:
			best = counts
			choices = append(choices[:0], owner)
		case comparison == 0:
			choices = append(choices, owner)
		default:
			// More loaded owners cannot enter the exact-tie choice set.
		}
	}
	if len(choices) == 0 {
		return "", false
	}
	return choose(choices), true
}

// assignment pairs one candidate PR with the owner chosen for it.
type assignment struct {
	number int
	owner  string
}

// decideAssignments is the pure decision core of the sweep. Given each owner's
// current load and the ordered candidate PRs, it picks the least-loaded owner
// for each PR (exact ties go through the random chooser), updates load as it
// goes so a batch spreads across owners, and honors the per-run cap. It
// performs no I/O; runSweep fetches the inputs and applies the output.
func decideAssignments(load map[string]ownerLoad, candidates []int, maxAssignments int, choose func([]string) string) []assignment {
	assignments := make([]assignment, 0, maxAssignments)
	for _, number := range candidates {
		if len(assignments) >= maxAssignments {
			break
		}
		owner, ok := chooseAssignee(load, choose)
		if !ok {
			continue
		}
		assignments = append(assignments, assignment{number: number, owner: owner})
		counts := load[owner]
		counts.open++
		load[owner] = counts
	}
	return assignments
}

type applyOutcome uint8

const (
	applySuccess applyOutcome = iota
	applySkipped              // the PR changed state (assigned, closed, draft) since the decision
	applyFailed               // transient error; the next run retries
)

// applyAssignment performs one decided assignment: it re-reads the PR to catch
// mid-sweep changes, adds the chosen owner, and, if the PR gained a competing
// assignee before the write landed, rolls back our owner so it does not stay
// double-assigned. Transient errors are wrapped and surfaced through the run's
// aggregate error so the root cause reaches the job annotation; the next run
// retries them. There is no marker label or batch abort: a failure only counts
// the PR as deferred.
func applyAssignment(ctx context.Context, client *github.Client, number int, owner string) (applyOutcome, error) {
	var current *github.Issue
	if err := retryableCall(ctx, func() error {
		var callErr error
		current, _, callErr = client.Issues.Get(ctx, repoOwner, repoName, number)
		return callErr
	}); err != nil {
		return applyFailed, fmt.Errorf("re-read PR #%d before assigning: %w", number, err)
	}
	if current == nil {
		// go-github returns a nil value for an empty or "null" 2xx body; do
		// not dereference it.
		return applyFailed, errors.New("re-read of PR returned an empty response")
	}
	if len(current.Assignees) > 0 {
		fmt.Printf("PR #%d was assigned elsewhere while the sweep ran; leaving it alone.\n", number)
		return applySkipped, nil
	}
	if (current.State != nil && *current.State != "open") || current.GetDraft() {
		fmt.Printf("PR #%d was closed or converted to draft while the sweep ran; leaving it alone.\n", number)
		return applySkipped, nil
	}
	issue, _, err := client.Issues.AddAssignees(ctx, repoOwner, repoName, number, []string{owner})
	if err != nil {
		return applyFailed, fmt.Errorf("add assignee: %w", err)
	}
	if issue == nil {
		// A nil value from an empty or "null" 2xx body: we cannot verify the
		// write landed, so treat it as a failure rather than dereference.
		return applyFailed, errors.New("assignment returned an empty response")
	}
	if len(issue.Assignees) > 1 {
		if err := removeAssignee(ctx, client, number, owner); err != nil {
			return applyFailed, fmt.Errorf("concurrent assignment detected; rollback failed: %w", err)
		}
		fmt.Printf("PR #%d gained another assignee during assignment; rolled back @%s.\n", number, owner)
		return applySkipped, nil
	}
	reflected := len(issue.Assignees) == 1 && strings.EqualFold(issue.Assignees[0].GetLogin(), owner)
	if !reflected {
		// No rollback here: a 2xx with zero or a different assignee means the
		// write did not apply our assignee on GitHub's side, so there is
		// nothing of ours to remove.
		return applyFailed, errors.New("assignment response did not reflect the selected owner")
	}
	fmt.Printf("Assigned @%s to PR #%d.\n", owner, number)
	return applySuccess, nil
}

func removeAssignee(ctx context.Context, client *github.Client, number int, assignee string) error {
	if err := retryableCall(ctx, func() error {
		var callErr error
		_, _, callErr = client.Issues.RemoveAssignees(ctx, repoOwner, repoName, number, []string{assignee})
		return callErr
	}); err != nil {
		return fmt.Errorf("remove assignee: %w", err)
	}
	return nil
}

// sweep supplies concrete production dependencies; runSweep receives explicit
// test values without time-zero or nil-function sentinel conventions.
func sweep(ctx context.Context, client *github.Client) error {
	return runSweep(ctx, client, time.Now().UTC(), func(choices []string) string {
		return choices[rand.IntN(len(choices))]
	})
}

// listOpenPRs fetches every open pull request, converting to the internal shape.
func listOpenPRs(ctx context.Context, client *github.Client) ([]pullRequest, error) {
	pulls, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.PullRequest, *github.Response, error) {
		return client.PullRequests.List(ctx, repoOwner, repoName, &github.PullRequestListOptions{State: "open", ListOptions: *opts})
	})
	if err != nil {
		return nil, err
	}
	result := make([]pullRequest, 0, len(pulls))
	for _, pr := range pulls {
		result = append(result, convertPullRequest(pr))
	}
	return result, nil
}

func convertPullRequest(pr *github.PullRequest) pullRequest {
	result := pullRequest{
		Number:    pr.GetNumber(),
		CreatedAt: pr.GetCreatedAt().Time,
		Draft:     pr.GetDraft(),
	}
	if pr.User != nil {
		result.User = &user{Login: pr.User.GetLogin(), Type: pr.User.GetType()}
	}
	for _, assignee := range pr.Assignees {
		result.Assignees = append(result.Assignees, user{Login: assignee.GetLogin()})
	}
	return result
}

// convertIssue adapts a search-result issue (closed PR payloads) to the
// internal pullRequest shape used by calculateLoad.
func convertIssue(issue *github.Issue) pullRequest {
	result := pullRequest{
		Number:    issue.GetNumber(),
		CreatedAt: issue.GetCreatedAt().Time,
	}
	if issue.User != nil {
		result.User = &user{Login: issue.User.GetLogin(), Type: issue.User.GetType()}
	}
	for _, assignee := range issue.Assignees {
		result.Assignees = append(result.Assignees, user{Login: assignee.GetLogin()})
	}
	return result
}

func runSweep(ctx context.Context, client *github.Client, now time.Time, choose func([]string) string) error {
	if len(defaultAssigneePool) == 0 {
		fmt.Println("::notice::The code owner pool is empty; add members in .github/scripts/consts.go.")
		return errors.New("code owner pool is empty")
	}

	// Open PRs once for load, then again for candidates: assignment can change
	// the pool state between the two passes, and the second fetch keeps the
	// per-PR decisions close to the assignments they make.
	openPRs, err := listOpenPRs(ctx, client)
	if err != nil {
		return fmt.Errorf("list open pull requests for load: %w", err)
	}
	since := now.Add(-closedLoadWindow).UTC().Format("2006-01-02T15:04:05Z")
	closedQuery := fmt.Sprintf("repo:%s is:pr is:closed closed:>=%s", defaultRepo, since)
	var totalClosed int
	var countResult *github.IssuesSearchResult
	if err := retryableCall(ctx, func() error {
		var callErr error
		countResult, _, callErr = client.Search.Issues(ctx, closedQuery, &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}})
		return callErr
	}); err != nil {
		return fmt.Errorf("count recently closed pull requests: %w", err)
	}
	totalClosed = countResult.GetTotal()
	if totalClosed > maxSearchResults {
		fmt.Printf("::warning::%d PRs closed in the last %d days, but the search API returns at most %d. Closed load is undercounted.\n", totalClosed, closedLoadDays, maxSearchResults)
	}
	closedIssues, err := eachPageBounded(ctx, func(opts *github.ListOptions) ([]*github.Issue, *github.Response, error) {
		result, resp, err := client.Search.Issues(ctx, closedQuery, &github.SearchOptions{ListOptions: *opts})
		if err != nil {
			return nil, resp, err
		}
		if result == nil {
			return nil, resp, errors.New("closed search returned an empty response")
		}
		return result.Issues, resp, nil
	}, maxSearchResults)
	if err != nil {
		return fmt.Errorf("list recently closed pull requests: %w", err)
	}
	closedPRs := make([]pullRequest, 0, len(closedIssues))
	for _, issue := range closedIssues {
		closedPRs = append(closedPRs, convertIssue(issue))
	}
	load, staleOwners := calculateLoad(openPRs, closedPRs, defaultAssigneePool, now, defaultOpenLoadMaxAgeDays)
	for _, owner := range slices.Sorted(maps.Keys(staleOwners)) {
		fmt.Printf("Open PRs assigned to %s older than %d days no longer count as load.\n", owner, defaultOpenLoadMaxAgeDays)
	}

	currentPRs, err := listOpenPRs(ctx, client)
	if err != nil {
		return fmt.Errorf("refresh pull request candidates: %w", err)
	}
	candidates := findCandidates(currentPRs, defaultAssigneePool)
	fmt.Printf("Found %d unassigned PR(s).\n", len(candidates))
	if len(candidates) == 0 {
		return nil
	}

	assignments := decideAssignments(load, candidates, defaultMaxAssignments, choose)
	// Give the apply phase its own deadline (see batchTimeout) so a GitHub
	// outage fails the run cleanly instead of draining every PR's retry budget
	// and being SIGTERM-killed mid-write at the workflow's timeout.
	applyCtx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()
	assigned, alreadyAssigned, failed := 0, 0, 0
	var firstErr error
	for _, a := range assignments {
		if err := applyCtx.Err(); err != nil {
			return fmt.Errorf("apply phase exceeded its %s budget: %w", batchTimeout, err)
		}
		outcome, err := applyAssignment(applyCtx, client, a.number, a.owner)
		if err != nil {
			fmt.Printf("::warning::Could not complete assignment of PR #%d to @%s: %v.\n", a.number, a.owner, err)
			if firstErr == nil {
				firstErr = err
			}
		}
		switch outcome {
		case applySuccess:
			assigned++
		case applySkipped:
			alreadyAssigned++
		case applyFailed:
			failed++
		default:
			// applyOutcome has exactly these three values; a new outcome must
			// be handled here and reflected in the summary below.
			return errors.New("unknown apply outcome")
		}
	}
	fmt.Printf("Done: assigned %d, %d already assigned or closed, %d deferred by errors.\n", assigned, alreadyAssigned, failed)
	// Surface a cancellation (e.g. SIGTERM) as such, not as an assignment
	// failure count; the root cause of a deferred batch must reach the job
	// annotation, not just a per-PR warning.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("sweep interrupted: %w", err)
	}
	if failed > 0 {
		if firstErr != nil {
			return fmt.Errorf("%d PR(s) could not be processed (first error: %v); they will be retried on the next run", failed, firstErr)
		}
		return fmt.Errorf("%d PR(s) could not be processed; they will be retried on the next run", failed)
	}
	return nil
}
