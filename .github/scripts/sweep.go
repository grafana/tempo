package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	// math/rand/v2 is fine for tie-breaking between equal-load owners; the
	// output is never used for security. nosemgrep: math-random-used
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v93/github"
)

type user struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

type label struct {
	Name string `json:"name"`
}

type pullRequest struct {
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"created_at"`
	State     string    `json:"state"`
	User      *user     `json:"user"`
	Assignees []user    `json:"assignees"`
	Draft     bool      `json:"draft"`
	Labels    []label   `json:"labels"`
}

type timelineEvent struct {
	Event string `json:"event"`
	Actor *user  `json:"actor"`
}

type candidate struct {
	number int
	marked bool
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
	return !hasLogin(owners, strings.ToLower(pr.User.Login))
}

func findCandidates(pulls []pullRequest, eligible []string, markerLabel string) []candidate {
	owners := make(map[string]struct{}, len(eligible))
	for _, owner := range eligible {
		owners[owner] = struct{}{}
	}
	byNumber := make(map[int]candidate, len(pulls))
	for _, pr := range pulls {
		if !candidateEligible(pr, owners) {
			continue
		}
		if _, ok := byNumber[pr.Number]; ok {
			continue
		}
		marked := false
		for _, label := range pr.Labels {
			// GitHub label identity is case-insensitive, so compare without case.
			if strings.EqualFold(label.Name, markerLabel) {
				marked = true
				break
			}
		}
		byNumber[pr.Number] = candidate{number: pr.Number, marked: marked}
	}
	result := make([]candidate, 0, len(byNumber))
	for _, candidate := range byNumber {
		result = append(result, candidate)
	}
	slices.SortFunc(result, func(a, b candidate) int { return cmp.Compare(a.number, b.number) })
	return result
}

func deliberatelyUnassigned(events []timelineEvent, botActor string) bool {
	lastBot, lastHuman := -1, -1
	for index, event := range events {
		actor := ""
		if event.Actor != nil {
			actor = event.Actor.Login
		}
		switch {
		case event.Event == "assigned" && actor == botActor:
			lastBot = index
		case event.Event == "unassigned" && actor != "" && actor != botActor:
			lastHuman = index
		default:
			// Other timeline events do not affect the latest assignment decision.
		}
	}
	return lastHuman > lastBot
}

func compareLoad(a, b ownerLoad) int {
	return cmp.Or(
		cmp.Compare(a.open+a.closed, b.open+b.closed),
		cmp.Compare(a.open, b.open),
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

func ensureMarkerLabel(ctx context.Context, client *github.Client, cfg config) error {
	owner, name, err := splitRepo(cfg.repo)
	if err != nil {
		return err
	}
	color := "ededed"
	description := "Assigned by the daily code owner sweep"
	err = retryableCall(ctx, func() error {
		var callErr error
		_, _, callErr = client.Issues.UpdateLabel(ctx, owner, name, cfg.markerLabel, github.UpdateIssueLabelRequest{Color: &color, Description: &description})
		return callErr
	})
	if err == nil {
		return nil
	}
	var statusError *github.ErrorResponse
	if !errors.As(err, &statusError) || statusError.Response.StatusCode != http.StatusNotFound {
		return fmt.Errorf("update marker label: %w", err)
	}
	if _, _, err := client.Issues.CreateLabel(ctx, owner, name, github.CreateIssueLabelRequest{Name: cfg.markerLabel, Color: &color, Description: &description}); err != nil {
		return fmt.Errorf("create marker label: %w", err)
	}
	return nil
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
	for _, candidate := range candidates {
		if len(assignments) >= maxAssignments {
			break
		}
		owner, ok := chooseAssignee(load, choose)
		if !ok {
			continue
		}
		assignments = append(assignments, assignment{number: candidate.number, owner: owner})
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
	applyAbort                // the marker label could not be applied; stop applying
)

// applyAssignment performs one decided assignment: it re-reads the PR to catch
// mid-sweep changes, adds the chosen owner, and applies the marker label,
// rolling back and aborting the batch when the marker cannot be applied.
func applyAssignment(ctx context.Context, client *github.Client, cfg config, number int, owner string) (applyOutcome, error) {
	repoOwner, repoName, err := splitRepo(cfg.repo)
	if err != nil {
		return applyFailed, err
	}
	var current *github.Issue
	if err := retryableCall(ctx, func() error {
		var callErr error
		current, _, callErr = client.Issues.Get(ctx, repoOwner, repoName, number)
		return callErr
	}); err != nil {
		fmt.Printf("::warning::Could not re-read PR #%d before assigning: %v.\n", number, err)
		return applyFailed, nil
	}
	if len(current.Assignees) > 0 {
		fmt.Printf("PR #%d was assigned elsewhere while the sweep ran; leaving it alone.\n", number)
		return applySkipped, nil
	}
	if (current.State != nil && *current.State != "open") || current.GetDraft() {
		fmt.Printf("PR #%d was closed or converted to draft while the sweep ran; leaving it alone.\n", number)
		return applySkipped, nil
	}
	alreadyAssigned, err := assignAndMark(ctx, client, cfg, number, owner)
	if err != nil {
		if errors.Is(err, errApplyMarker) {
			return applyAbort, err
		}
		return applyFailed, err
	}
	if alreadyAssigned {
		fmt.Printf("PR #%d gained another assignee during assignment; rolled back @%s.\n", number, owner)
		return applySkipped, nil
	}
	fmt.Printf("Assigned @%s to PR #%d.\n", owner, number)
	return applySuccess, nil
}

// errApplyMarker distinguishes marker-application failures from assignment
// failures: a marker failure means every later candidate would also assign and
// roll back, so the sweep aborts instead of churning the whole batch.
var errApplyMarker = errors.New("apply marker label")

// assignAndMark rolls back only this sweep's assignee when another owner races
// the write or the marker cannot be applied, avoiding silent partial assignments.
func assignAndMark(ctx context.Context, client *github.Client, cfg config, number int, assignee string) (bool, error) {
	owner, name, err := splitRepo(cfg.repo)
	if err != nil {
		return false, err
	}
	issue, _, err := client.Issues.AddAssignees(ctx, owner, name, number, []string{assignee})
	if err != nil {
		return false, fmt.Errorf("add assignee: %w", err)
	}
	if len(issue.Assignees) > 1 {
		if err := removeAssignee(ctx, client, cfg, number, assignee); err != nil {
			return false, fmt.Errorf("concurrent assignment detected; rollback failed: %w", err)
		}
		return true, nil
	}
	reflected := len(issue.Assignees) == 1 && strings.EqualFold(issue.Assignees[0].GetLogin(), assignee)
	if !reflected {
		// No rollback here: a 2xx with zero or a different assignee means the
		// write did not apply our assignee on GitHub's side, so there is
		// nothing of ours to remove.
		return false, errors.New("assignment response did not reflect the selected owner")
	}
	if _, _, err := client.Issues.AddLabelsToIssue(ctx, owner, name, number, []string{cfg.markerLabel}); err != nil {
		markerErr := fmt.Errorf("%w: %v", errApplyMarker, err)
		if rollbackErr := removeAssignee(ctx, client, cfg, number, assignee); rollbackErr != nil {
			return false, errors.Join(markerErr, fmt.Errorf("rollback assignee: %w", rollbackErr))
		}
		return false, markerErr
	}
	return false, nil
}

func removeAssignee(ctx context.Context, client *github.Client, cfg config, number int, assignee string) error {
	owner, name, err := splitRepo(cfg.repo)
	if err != nil {
		return err
	}
	if err := retryableCall(ctx, func() error {
		var callErr error
		_, _, callErr = client.Issues.RemoveAssignees(ctx, owner, name, number, []string{assignee})
		return callErr
	}); err != nil {
		return fmt.Errorf("remove assignee: %w", err)
	}
	return nil
}

// sweep supplies concrete production dependencies; runSweep receives explicit
// test values without time-zero or nil-function sentinel conventions.
func sweep(ctx context.Context, client *github.Client, cfg config) error {
	return runSweep(ctx, client, cfg, time.Now().UTC(), func(choices []string) string {
		return choices[rand.IntN(len(choices))]
	})
}

// listOpenPRs fetches every open pull request, converting to the internal shape.
func listOpenPRs(ctx context.Context, client *github.Client, repo string) ([]pullRequest, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return nil, err
	}
	pulls, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.PullRequest, *github.Response, error) {
		return client.PullRequests.List(ctx, owner, name, &github.PullRequestListOptions{State: "open", ListOptions: *opts})
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
		State:     pr.GetState(),
		Draft:     pr.GetDraft(),
	}
	if pr.User != nil {
		result.User = &user{Login: pr.User.GetLogin(), Type: pr.User.GetType()}
	}
	for _, assignee := range pr.Assignees {
		result.Assignees = append(result.Assignees, user{Login: assignee.GetLogin()})
	}
	for _, l := range pr.Labels {
		result.Labels = append(result.Labels, label{Name: l.GetName()})
	}
	return result
}

// convertIssue adapts a search-result issue (closed PR payloads) to the
// internal pullRequest shape used by calculateLoad.
func convertIssue(issue *github.Issue) pullRequest {
	result := pullRequest{
		Number:    issue.GetNumber(),
		CreatedAt: issue.GetCreatedAt().Time,
		State:     issue.GetState(),
		Draft:     issue.GetDraft(),
	}
	if issue.User != nil {
		result.User = &user{Login: issue.User.GetLogin(), Type: issue.User.GetType()}
	}
	for _, assignee := range issue.Assignees {
		result.Assignees = append(result.Assignees, user{Login: assignee.GetLogin()})
	}
	return result
}

func convertTimeline(events []*github.Timeline) []timelineEvent {
	result := make([]timelineEvent, 0, len(events))
	for _, event := range events {
		converted := timelineEvent{Event: event.GetEvent()}
		if event.Actor != nil {
			converted.Actor = &user{Login: event.Actor.GetLogin()}
		}
		result = append(result, converted)
	}
	return result
}

func runSweep(ctx context.Context, client *github.Client, cfg config, now time.Time, choose func([]string) string) error {
	repoOwner, repoName, err := splitRepo(cfg.repo)
	if err != nil {
		return err
	}
	eligible, err := resolvePool(ctx, client, cfg)
	if err != nil {
		return err
	}
	if len(eligible) == 0 {
		fmt.Println("::notice::The code owner pool resolved to nobody; check .github/scripts/pool.go and that pool members are assignable in the repository.")
		return fmt.Errorf("no assignable code owner was found")
	}
	fmt.Printf("Eligible owners: %d.\n", len(eligible))

	// Open PRs once for load, then again for candidates: assignment can change
	// the pool state between the two passes, and the second fetch keeps the
	// per-PR decisions close to the assignments they make.
	openPRs, err := listOpenPRs(ctx, client, cfg.repo)
	if err != nil {
		return fmt.Errorf("list open pull requests for load: %w", err)
	}
	since := now.Add(-closedLoadWindow).UTC().Format("2006-01-02T15:04:05Z")
	closedQuery := fmt.Sprintf("repo:%s is:pr is:closed closed:>=%s", cfg.repo, since)
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
		return result.Issues, resp, nil
	}, maxSearchResults)
	if err != nil {
		return fmt.Errorf("list recently closed pull requests: %w", err)
	}
	closedPRs := make([]pullRequest, 0, len(closedIssues))
	for _, issue := range closedIssues {
		closedPRs = append(closedPRs, convertIssue(issue))
	}
	load, staleOwners := calculateLoad(openPRs, closedPRs, eligible, now, cfg.openLoadMaxAgeDays)
	for _, owner := range slices.Sorted(maps.Keys(staleOwners)) {
		fmt.Printf("Open PRs assigned to %s older than %d days no longer count as load.\n", owner, cfg.openLoadMaxAgeDays)
	}

	currentPRs, err := listOpenPRs(ctx, client, cfg.repo)
	if err != nil {
		return fmt.Errorf("refresh pull request candidates: %w", err)
	}
	candidates := findCandidates(currentPRs, eligible, cfg.markerLabel)
	fmt.Printf("Found %d unassigned PR(s).\n", len(candidates))
	if len(candidates) == 0 {
		return nil
	}
	// Marked candidates need their timeline read before the pure decision: a
	// PR a human deliberately unassigned after a bot assignment is excluded.
	active := make([]candidate, 0, len(candidates))
	deliberatelySkipped, failed := 0, 0
	for _, candidate := range candidates {
		if !candidate.marked {
			active = append(active, candidate)
			continue
		}
		events, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.Timeline, *github.Response, error) {
			return client.Issues.ListIssueTimeline(ctx, repoOwner, repoName, candidate.number, opts)
		})
		if err != nil {
			fmt.Printf("::warning::Could not read the timeline for PR #%d: %v; leaving it for the next run.\n", candidate.number, err)
			failed++
			continue
		}
		if deliberatelyUnassigned(convertTimeline(events), cfg.botActor) {
			fmt.Printf("PR #%d was auto-assigned before and unassigned by a person; skipping.\n", candidate.number)
			deliberatelySkipped++
			continue
		}
		active = append(active, candidate)
	}
	// The marker label is load-bearing: without it, later runs cannot tell a
	// deliberate human unassignment from a PR that was never assigned. Fail
	// the run rather than assigning unmarked PRs.
	if err := ensureMarkerLabel(ctx, client, cfg); err != nil {
		return fmt.Errorf("ensure marker label: %w", err)
	}

	assignments := decideAssignments(load, active, cfg.maxAssignments, choose)
	assigned, alreadyAssigned := 0, 0
	for _, a := range assignments {
		outcome, err := applyAssignment(ctx, client, cfg, a.number, a.owner)
		if err != nil && outcome != applyAbort {
			fmt.Printf("::warning::Could not complete assignment of PR #%d to @%s: %v.\n", a.number, a.owner, err)
		}
		switch outcome {
		case applySuccess:
			assigned++
		case applySkipped:
			alreadyAssigned++
		case applyFailed:
			failed++
		case applyAbort:
			failed++
			return fmt.Errorf("PR #%d was rolled back because the %q marker label could not be applied; remaining PRs wait for the next run", a.number, cfg.markerLabel)
		default:
			// applyOutcome has exactly these four values; a new outcome must
			// be handled here and reflected in the summary below.
			return errors.New("unknown apply outcome")
		}
	}
	fmt.Printf("Done: assigned %d, skipped %d deliberately unassigned, %d already assigned or closed, %d deferred by errors.\n", assigned, deliberatelySkipped, alreadyAssigned, failed)
	if failed > 0 {
		return fmt.Errorf("%d PR(s) could not be processed; they will be retried on the next run", failed)
	}
	return nil
}
