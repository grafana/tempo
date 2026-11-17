package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
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
	_, _, err = client.Issues.UpdateLabel(ctx, owner, name, cfg.markerLabel, github.UpdateIssueLabelRequest{Color: &color, Description: &description})
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

type candidateOutcome uint8

const (
	candidateAssigned candidateOutcome = iota
	candidateDeliberatelySkipped
	candidateAlreadyAssigned
	candidateFailed
)

// processCandidate handles all per-PR decisions separately from batch/load orchestration.
func processCandidate(
	ctx context.Context,
	client *github.Client,
	cfg config,
	candidate candidate,
	load map[string]ownerLoad,
	choose func([]string) string,
) candidateOutcome {
	owner, name, err := splitRepo(cfg.repo)
	if err != nil {
		return candidateFailed
	}
	if candidate.marked {
		events, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.Timeline, *github.Response, error) {
			return client.Issues.ListIssueTimeline(ctx, owner, name, candidate.number, opts)
		})
		if err != nil {
			fmt.Printf("::warning::Could not read the timeline for PR #%d: %v; leaving it for the next run.\n", candidate.number, err)
			return candidateFailed
		}
		if deliberatelyUnassigned(convertTimeline(events), cfg.botActor) {
			fmt.Printf("PR #%d was auto-assigned before and unassigned by a person; skipping.\n", candidate.number)
			return candidateDeliberatelySkipped
		}
	}

	current, _, err := client.Issues.Get(ctx, owner, name, candidate.number)
	if err != nil {
		fmt.Printf("::warning::Could not re-read PR #%d before assigning: %v.\n", candidate.number, err)
		return candidateFailed
	}
	if len(current.Assignees) > 0 {
		fmt.Printf("PR #%d was assigned elsewhere while the sweep ran; leaving it alone.\n", candidate.number)
		return candidateAlreadyAssigned
	}
	if (current.State != nil && *current.State != "open") || current.GetDraft() {
		fmt.Printf("PR #%d was closed or converted to draft while the sweep ran; leaving it alone.\n", candidate.number)
		return candidateAlreadyAssigned
	}
	assignee, ok := chooseAssignee(load, choose)
	if !ok {
		fmt.Printf("::warning::No eligible code owner is available for PR #%d.\n", candidate.number)
		return candidateFailed
	}

	alreadyAssigned, err := assignAndMark(ctx, client, cfg, candidate.number, assignee)
	if err != nil {
		fmt.Printf("::warning::Could not complete assignment of PR #%d to @%s: %v.\n", candidate.number, assignee, err)
		return candidateFailed
	}
	if alreadyAssigned {
		fmt.Printf("PR #%d gained another assignee during assignment; rolled back @%s.\n", candidate.number, assignee)
		return candidateAlreadyAssigned
	}
	counts := load[assignee]
	counts.open++
	load[assignee] = counts
	fmt.Printf("Assigned @%s to PR #%d (load: %d = %d open + %d closed in the past %d days).\n", assignee, candidate.number, counts.open+counts.closed, counts.open, counts.closed, closedLoadDays)
	return candidateAssigned
}

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
		markerErr := fmt.Errorf("apply marker label: %w", err)
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
	if _, _, err := client.Issues.RemoveAssignees(ctx, owner, name, number, []string{assignee}); err != nil {
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
	codeowners, err := readCodeowners(ctx, client, cfg.repo)
	if err != nil {
		return err
	}
	groups := parseCodeowners(codeowners, cfg.excludedOwners)
	printOwnerNotice("Excluded from the pool by EXCLUDED_OWNERS", groups.excluded)
	printOwnerNotice("These CODEOWNERS owners appear in only some rules and are excluded from the pool", groups.partial)

	eligible, err := resolvePool(ctx, client, cfg, groups)
	if err != nil {
		return err
	}
	if len(eligible) == 0 {
		fmt.Println("::notice::Restore ASSIGNEE_POOL, configure CODEOWNER_ASSIGNEES, or provide org team-read permission.")
		return fmt.Errorf("no assignable code owner was found (users %v, teams %v, partial %v)",
			slices.Sorted(maps.Keys(groups.users)), slices.Sorted(maps.Keys(groups.teams)), slices.Sorted(maps.Keys(groups.partial)))
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
	if result, _, err := client.Search.Issues(ctx, closedQuery, &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}}); err != nil {
		return fmt.Errorf("count recently closed pull requests: %w", err)
	} else {
		totalClosed = result.GetTotal()
	}
	if totalClosed > maxSearchResults {
		fmt.Printf("::warning::%d PRs closed in the last %d days, but the search API returns at most %d. Closed load is undercounted.\n", totalClosed, closedLoadDays, maxSearchResults)
	}
	closedIssues, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.Issue, *github.Response, error) {
		result, resp, err := client.Search.Issues(ctx, closedQuery, &github.SearchOptions{ListOptions: *opts})
		if err != nil {
			return nil, resp, err
		}
		return result.Issues, resp, nil
	})
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
	// The marker label is load-bearing: without it, later runs cannot tell a
	// deliberate human unassignment from a PR that was never assigned. Fail
	// the run rather than assigning unmarked PRs.
	if err := ensureMarkerLabel(ctx, client, cfg); err != nil {
		return fmt.Errorf("ensure marker label: %w", err)
	}

	assigned, deliberatelySkipped, alreadyAssigned, failed := 0, 0, 0, 0
	for _, candidate := range candidates {
		if assigned >= cfg.maxAssignments {
			fmt.Printf("Reached the per-run cap of %d; the rest are left for the next run.\n", cfg.maxAssignments)
			break
		}
		switch processCandidate(ctx, client, cfg, candidate, load, choose) {
		case candidateAssigned:
			assigned++
		case candidateDeliberatelySkipped:
			deliberatelySkipped++
		case candidateAlreadyAssigned:
			alreadyAssigned++
		case candidateFailed:
			failed++
		default:
			// candidateOutcome has exactly these four values; a new outcome
			// must be handled here and reflected in the summary below.
			return errors.New("unknown candidate outcome")
		}
	}
	fmt.Printf("Done: assigned %d, skipped %d deliberately unassigned, %d already assigned, %d deferred by errors.\n", assigned, deliberatelySkipped, alreadyAssigned, failed)
	if failed > 0 {
		return fmt.Errorf("%d PR(s) could not be processed; they will be retried on the next run", failed)
	}
	return nil
}

func printOwnerNotice(message string, owners map[string]struct{}) {
	if len(owners) > 0 {
		fmt.Printf("::notice::%s: %s\n", message, strings.Join(slices.Sorted(maps.Keys(owners)), " "))
	}
}
