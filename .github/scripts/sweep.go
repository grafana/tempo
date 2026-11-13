package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"
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
	User      *user     `json:"user"`
	Assignees []user    `json:"assignees"`
	Draft     bool      `json:"draft"`
	Labels    []label   `json:"labels"`
}

type timelineEvent struct {
	Event string `json:"event"`
	Actor *user  `json:"actor"`
}

type issue struct {
	Number int `json:"number"`
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
				if state.kind == openLoad && now.Sub(pr.CreatedAt) > maxOpenAge {
					staleOwners[owner] = struct{}{}
					continue
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
			if label.Name == markerLabel {
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

// randomChoice returns empty for an empty set and otherwise uses the supplied
// random index function. Injection makes the production choice fully testable.
func randomChoice(choices []string, intN func(int) int) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[intN(len(choices))]
}

func ensureMarkerLabel(ctx context.Context, client *githubClient, cfg config) error {
	body := map[string]string{
		"color":       "ededed",
		"description": "Assigned by the daily code owner sweep",
	}
	_, err := client.request(ctx, http.MethodPatch, repositoryEndpoint(cfg.repo, "/labels/"+url.PathEscape(cfg.markerLabel)), nil, body, nil)
	if err == nil {
		return nil
	}
	var statusError *apiError
	if !errors.As(err, &statusError) || statusError.status != http.StatusNotFound {
		return fmt.Errorf("update marker label: %w", err)
	}
	body["name"] = cfg.markerLabel
	if _, err := client.request(ctx, http.MethodPost, repositoryEndpoint(cfg.repo, "/labels"), nil, body, nil); err != nil {
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
	client *githubClient,
	cfg config,
	candidate candidate,
	load map[string]ownerLoad,
	choose func([]string) string,
) candidateOutcome {
	if candidate.marked {
		events, err := paginateList[timelineEvent](ctx, client, issueEndpoint(cfg.repo, candidate.number, "/timeline"), pageQuery())
		if err != nil {
			fmt.Printf("::warning::Could not read the timeline for PR #%d: %v; leaving it for the next run.\n", candidate.number, err)
			return candidateFailed
		}
		if deliberatelyUnassigned(events, cfg.botActor) {
			fmt.Printf("PR #%d was auto-assigned before and unassigned by a person; skipping.\n", candidate.number)
			return candidateDeliberatelySkipped
		}
	}

	var current struct {
		Assignees []user `json:"assignees"`
	}
	if _, err := client.request(ctx, http.MethodGet, issueEndpoint(cfg.repo, candidate.number, ""), nil, nil, &current); err != nil {
		fmt.Printf("::warning::Could not re-read PR #%d before assigning: %v.\n", candidate.number, err)
		return candidateFailed
	}
	if len(current.Assignees) > 0 {
		fmt.Printf("PR #%d was assigned elsewhere while the sweep ran; leaving it alone.\n", candidate.number)
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
func assignAndMark(ctx context.Context, client *githubClient, cfg config, number int, assignee string) (bool, error) {
	var response pullRequest
	payload := map[string][]string{"assignees": {assignee}}
	if _, err := client.request(ctx, http.MethodPost, issueEndpoint(cfg.repo, number, "/assignees"), nil, payload, &response); err != nil {
		return false, fmt.Errorf("add assignee: %w", err)
	}
	if len(response.Assignees) > 1 {
		if err := removeAssignee(ctx, client, cfg, number, assignee); err != nil {
			return false, fmt.Errorf("concurrent assignment detected; rollback failed: %w", err)
		}
		return true, nil
	}
	reflected := len(response.Assignees) == 1 && strings.EqualFold(response.Assignees[0].Login, assignee)
	if !reflected {
		return false, errors.New("assignment response did not reflect the selected owner")
	}
	if _, err := client.request(ctx, http.MethodPost, issueEndpoint(cfg.repo, number, "/labels"), nil, map[string][]string{"labels": {cfg.markerLabel}}, nil); err != nil {
		markerErr := fmt.Errorf("apply marker label: %w", err)
		if rollbackErr := removeAssignee(ctx, client, cfg, number, assignee); rollbackErr != nil {
			return false, errors.Join(markerErr, fmt.Errorf("rollback assignee: %w", rollbackErr))
		}
		return false, markerErr
	}
	return false, nil
}

func removeAssignee(ctx context.Context, client *githubClient, cfg config, number int, assignee string) error {
	_, err := client.request(ctx, http.MethodDelete, issueEndpoint(cfg.repo, number, "/assignees"), nil, map[string][]string{"assignees": {assignee}}, nil)
	return err
}

// sweep supplies concrete production dependencies; runSweep receives explicit
// test values without time-zero or nil-function sentinel conventions.
func sweep(ctx context.Context, client *githubClient, cfg config) error {
	return runSweep(ctx, client, cfg, time.Now().UTC(), func(choices []string) string {
		return randomChoice(choices, rand.IntN)
	})
}

func runSweep(ctx context.Context, client *githubClient, cfg config, now time.Time, choose func([]string) string) error {
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

	openPRs, err := paginateList[pullRequest](ctx, client, repositoryEndpoint(cfg.repo, "/pulls"), withQuery(pageQuery(), "state", "open"))
	if err != nil {
		return fmt.Errorf("list open pull requests for load: %w", err)
	}
	since := now.Add(-closedLoadWindow).UTC().Format("2006-01-02T15:04:05Z")
	closedQuery := fmt.Sprintf("repo:%s is:pr is:closed closed:>=%s", cfg.repo, since)
	var search struct {
		TotalCount int `json:"total_count"`
	}
	if _, err := client.request(ctx, http.MethodGet, "search/issues", queryWith("q", closedQuery, "per_page", "1"), nil, &search); err != nil {
		return fmt.Errorf("count recently closed pull requests: %w", err)
	}
	if search.TotalCount > 1000 {
		fmt.Printf("::warning::%d PRs closed in the last %d days, but the search API returns at most 1000. Closed load is undercounted.\n", search.TotalCount, closedLoadDays)
	}
	closedPRs, err := paginateSearch[pullRequest](ctx, client, "search/issues", withQuery(pageQuery(), "q", closedQuery))
	if err != nil {
		return fmt.Errorf("list recently closed pull requests: %w", err)
	}
	load, staleOwners := calculateLoad(openPRs, closedPRs, eligible, now, cfg.openLoadMaxAgeDays)
	for _, owner := range slices.Sorted(maps.Keys(staleOwners)) {
		fmt.Fprintf(os.Stderr, "Open PRs by %s older than %d days no longer count as load.\n", owner, cfg.openLoadMaxAgeDays)
	}

	currentPRs, err := paginateList[pullRequest](ctx, client, repositoryEndpoint(cfg.repo, "/pulls"), withQuery(pageQuery(), "state", "open"))
	if err != nil {
		return fmt.Errorf("refresh pull request candidates: %w", err)
	}
	candidates := findCandidates(currentPRs, eligible, cfg.markerLabel)
	fmt.Printf("Found %d unassigned PR(s).\n", len(candidates))
	if len(candidates) == 0 {
		return nil
	}
	if err := ensureMarkerLabel(ctx, client, cfg); err != nil {
		fmt.Printf("::warning::Could not ensure the %s label exists: %v.\n", cfg.markerLabel, err)
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

func withQuery(values url.Values, key, value string) url.Values {
	values.Set(key, value)
	return values
}
