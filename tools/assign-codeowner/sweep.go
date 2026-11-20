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
	Login string
	Type  string
}

type pullRequest struct {
	Number    int
	CreatedAt time.Time
	User      *user
	Assignees []user
	Draft     bool
	// Weight is the PR's load contribution (1..5) from its non-vendored diff
	// size; see weight.go. Zero is treated as 1.
	Weight int
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
					counts.open += weightOrDefault(pr.Weight)
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

// candidate is an eligible unassigned PR plus the load weight it will add to
// whichever owner receives it.
type candidate struct {
	number int
	weight int
}

func findCandidates(pulls []pullRequest, eligible []string) []candidate {
	owners := make(map[string]struct{}, len(eligible))
	for _, owner := range eligible {
		owners[owner] = struct{}{}
	}
	result := make([]candidate, 0, len(pulls))
	for _, pr := range pulls {
		if candidateEligible(pr, owners) {
			result = append(result, candidate{number: pr.Number, weight: weightOrDefault(pr.Weight)})
		}
	}
	slices.SortFunc(result, func(a, b candidate) int { return cmp.Compare(a.number, b.number) })
	return result
}

// compareLoad orders owners by open load (the size-weighted sum of assigned
// open PRs is the primary signal of how busy an owner is), breaking ties by
// recently-closed count so an owner who just shed a batch stays at the back of
// the queue. The recently-closed count must never outweigh the open load.
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
func decideAssignments(load map[string]ownerLoad, candidates []candidate, maxAssignments int, choose func([]string) string) []assignment {
	assignments := make([]assignment, 0, maxAssignments)
	for _, c := range candidates {
		if len(assignments) >= maxAssignments {
			break
		}
		owner, ok := chooseAssignee(load, choose)
		if !ok {
			continue
		}
		assignments = append(assignments, assignment{number: c.number, owner: owner})
		// Charge the receiving owner the PR's weight so a heavy PR pushes the
		// next decision toward a less-loaded owner.
		counts := load[owner]
		counts.open += c.weight
		load[owner] = counts
	}
	return assignments
}

// applyAssignment adds the chosen owner to the PR. AddAssignees is additive and
// idempotent, so a PR that changed since the decision (already assigned, closed,
// or converted to draft) is handled by the API rather than pre-checked, and the
// 2xx response is not inspected: there is no rollback logic for a concurrent
// assignee, so we trust the write and surface only hard failures.
func applyAssignment(ctx context.Context, client *github.Client, number int, owner string) error {
	if _, _, err := client.Issues.AddAssignees(ctx, repoOwner, repoName, number, []string{owner}); err != nil {
		return fmt.Errorf("add assignee to PR #%d: %w", number, err)
	}
	fmt.Printf("Assigned @%s to PR #%d.\n", owner, number)
	return nil
}

// sweep supplies concrete production dependencies; runSweep receives explicit
// test values without time-zero or nil-function sentinel conventions.
func sweep(ctx context.Context, client *github.Client) error {
	return runSweep(ctx, client, time.Now().UTC(), func(choices []string) string {
		return choices[rand.IntN(len(choices))]
	})
}

// listOpenPRs fetches every open pull request, converting to the internal shape
// and computing each PR's size weight. The weight costs one extra paginated
// request per PR; a failed size lookup degrades to weight 1 rather than
// aborting the sweep.
func listOpenPRs(ctx context.Context, client *github.Client) ([]pullRequest, error) {
	pulls, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.PullRequest, *github.Response, error) {
		return client.PullRequests.List(ctx, repoOwner, repoName, &github.PullRequestListOptions{State: "open", ListOptions: *opts})
	})
	if err != nil {
		return nil, err
	}
	result := make([]pullRequest, 0, len(pulls))
	for _, pr := range pulls {
		converted := convertPullRequest(pr)
		weight, weightErr := pullWeight(ctx, client, converted.Number)
		if weightErr != nil {
			fmt.Printf("::warning::Could not compute the size weight for PR #%d (assuming 1): %v.\n", converted.Number, weightErr)
			weight = 1
		}
		converted.Weight = weight
		result = append(result, converted)
	}
	return result, nil
}

func convertPullRequest(pr *github.PullRequest) pullRequest {
	if pr == nil {
		return pullRequest{}
	}
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
	if issue == nil {
		return pullRequest{}
	}
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

// runSweep orchestrates the daily sweep as three focused stages — build the
// load snapshot, plan the assignments, then apply them — so each stage is a
// small, individually reviewable function.
func runSweep(ctx context.Context, client *github.Client, now time.Time, choose func([]string) string) error {
	if len(defaultAssigneePool) == 0 {
		fmt.Println("::notice::The code owner pool is empty; add members in tools/assign-codeowner/consts.go.")
		return errors.New("code owner pool is empty")
	}

	openPRs, load, err := loadSnapshot(ctx, client, now)
	if err != nil {
		return err
	}
	assignments := planAssignments(openPRs, load, choose)
	if len(assignments) == 0 {
		return nil
	}
	return applyAssignments(ctx, client, assignments)
}

// loadSnapshot fetches the open PRs and the PRs closed within closedLoadWindow,
// computes each pool owner's load, and reports stale (over-maxOpenAgeDays) open
// PRs. It also returns the open-PR snapshot so candidate selection reuses the
// same fetch instead of hitting the API twice.
func loadSnapshot(ctx context.Context, client *github.Client, now time.Time) ([]pullRequest, map[string]ownerLoad, error) {
	openPRs, err := listOpenPRs(ctx, client)
	if err != nil {
		return nil, nil, fmt.Errorf("list open pull requests: %w", err)
	}
	closedPRs, err := recentlyClosedPRs(ctx, client, now)
	if err != nil {
		return nil, nil, err
	}
	load, staleOwners := calculateLoad(openPRs, closedPRs, defaultAssigneePool, now, defaultOpenLoadMaxAgeDays)
	for _, owner := range slices.Sorted(maps.Keys(staleOwners)) {
		fmt.Printf("Open PRs assigned to %s older than %d days no longer count as load.\n", owner, defaultOpenLoadMaxAgeDays)
	}
	return openPRs, load, nil
}

// recentlyClosedPRs fetches the PRs closed within closedLoadWindow, converted
// to internal shape. It warns when the search API's 1000-result cap truncates
// the set, in which case closed load is undercounted.
func recentlyClosedPRs(ctx context.Context, client *github.Client, now time.Time) ([]pullRequest, error) {
	closedQuery := fmt.Sprintf("repo:%s is:pr is:closed closed:>=%s", defaultRepo, now.Add(-closedLoadWindow).UTC().Format("2006-01-02T15:04:05Z"))
	count, _, err := client.Search.Issues(ctx, closedQuery, &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}})
	if err != nil {
		return nil, fmt.Errorf("count recently closed pull requests: %w", err)
	}
	if count == nil {
		return nil, errors.New("closed count returned an empty response")
	}
	total := count.GetTotal()
	if total > maxSearchResults {
		fmt.Printf("::warning::%d PRs closed in the last %d days, but the search API returns at most %d. Closed load is undercounted.\n", total, closedLoadDays, maxSearchResults)
	}
	issues, err := eachPageBounded(ctx, func(opts *github.ListOptions) ([]*github.Issue, *github.Response, error) {
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
		return nil, fmt.Errorf("list recently closed pull requests: %w", err)
	}
	closed := make([]pullRequest, 0, len(issues))
	for _, issue := range issues {
		closed = append(closed, convertIssue(issue))
	}
	return closed, nil
}

// planAssignments selects the unassigned candidates from the open-PR snapshot
// and decides which owners get which PRs this run, honoring the per-run cap. An
// empty result means nothing to assign this cycle.
func planAssignments(openPRs []pullRequest, load map[string]ownerLoad, choose func([]string) string) []assignment {
	candidates := findCandidates(openPRs, defaultAssigneePool)
	fmt.Printf("Found %d unassigned PR(s).\n", len(candidates))
	return decideAssignments(load, candidates, defaultMaxAssignments, choose)
}

// applyAssignments runs the decided assignments and returns an aggregate error
// carrying the first failure's root cause, or the cancellation itself, so the
// job annotation stays actionable.
func applyAssignments(ctx context.Context, client *github.Client, assignments []assignment) error {
	assigned, failed := 0, 0
	var firstErr error
	for _, a := range assignments {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("sweep interrupted: %w", err)
		}
		if err := applyAssignment(ctx, client, a.number, a.owner); err != nil {
			failed++
			fmt.Printf("::warning::Could not complete assignment of PR #%d to @%s: %v.\n", a.number, a.owner, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		assigned++
	}
	fmt.Printf("Done: assigned %d, %d deferred by errors.\n", assigned, failed)
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
