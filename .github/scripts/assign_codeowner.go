// Command assign_codeowner assigns unassigned pull requests using recent review load.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const apiVersion = "2022-11-28"

var codeownersPaths = []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"}

type apiError struct {
	method string
	path   string
	status int
	detail string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s %s failed with HTTP %d: %s", e.method, e.path, e.status, e.detail)
}

type githubClient struct {
	token      string
	baseURL    string
	httpClient *http.Client
	sleep      func(context.Context, time.Duration) error
}

func newGitHubClient(token string) *githubClient {
	return &githubClient{
		token:      token,
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		sleep:      sleepContext,
	}
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *githubClient) request(
	ctx context.Context,
	method, path string,
	query url.Values,
	body any,
	out any,
) (http.Header, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode %s %s: %w", method, path, err)
		}
	}

	for attempt := 1; attempt <= 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		headers, err := c.requestOnce(ctx, method, path, query, payload, out)
		if err == nil {
			return headers, nil
		}
		if attempt == 3 {
			return nil, err
		}
		if err := c.sleep(ctx, time.Duration(attempt*2)*time.Second); err != nil {
			return nil, err
		}
	}
	panic("unreachable")
}

func (c *githubClient) requestOnce(
	ctx context.Context,
	method, path string,
	query url.Values,
	payload []byte,
	out any,
) (http.Header, error) {
	requestURL := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		requestURL = strings.TrimSuffix(c.baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
	}
	if len(query) > 0 {
		separator := "?"
		if strings.Contains(requestURL, "?") {
			separator = "&"
		}
		requestURL += separator + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("build %s %s: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "tempo-codeowner-sweep")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, &apiError{method: method, path: path, status: resp.StatusCode, detail: string(detail)}
	}
	if out == nil {
		_, err = io.Copy(io.Discard, resp.Body)
	} else {
		err = json.NewDecoder(resp.Body).Decode(out)
	}
	if err != nil {
		return nil, fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return resp.Header, nil
}

func paginate[T any](
	ctx context.Context,
	client *githubClient,
	path string,
	query url.Values,
	wrappedItems bool,
) ([]T, error) {
	var results []T
	for path != "" {
		var headers http.Header
		var err error
		if wrappedItems {
			var page struct {
				Items []T `json:"items"`
			}
			headers, err = client.request(ctx, http.MethodGet, path, query, nil, &page)
			results = append(results, page.Items...)
		} else {
			var page []T
			headers, err = client.request(ctx, http.MethodGet, path, query, nil, &page)
			results = append(results, page...)
		}
		if err != nil {
			return nil, err
		}
		query = nil
		path = nextPage(headers.Get("Link"))
	}
	return results, nil
}

func nextPage(header string) string {
	for _, link := range strings.Split(header, ",") {
		parts := strings.Split(link, ";")
		if len(parts) < 2 {
			continue
		}
		for _, parameter := range parts[1:] {
			if strings.TrimSpace(parameter) == `rel="next"` {
				return strings.Trim(strings.TrimSpace(parts[0]), "<>")
			}
		}
	}
	return ""
}

type config struct {
	repo               string
	assigneePool       string
	excludedOwners     string
	maxAssignments     int
	openLoadMaxAgeDays int
	markerLabel        string
	botActor           string
}

func configFromEnv() (config, error) {
	repo, err := requiredEnv("REPO")
	if err != nil {
		return config{}, err
	}
	maxAssignments, err := envInt("MAX_ASSIGNMENTS", 25)
	if err != nil {
		return config{}, err
	}
	maxOpenAge, err := envInt("OPEN_LOAD_MAX_AGE_DAYS", 90)
	if err != nil {
		return config{}, err
	}
	return config{
		repo:               repo,
		assigneePool:       os.Getenv("ASSIGNEE_POOL"),
		excludedOwners:     os.Getenv("EXCLUDED_OWNERS"),
		maxAssignments:     maxAssignments,
		openLoadMaxAgeDays: maxOpenAge,
		markerLabel:        envDefault("MARKER_LABEL", "codeowner-sweep"),
		botActor:           envDefault("BOT_ACTOR", "github-actions[bot]"),
	}, nil
}

func requiredEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	return value, nil
}

func envDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value := envDefault(name, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

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
	author string
	marked bool
}

type ownerLoad struct {
	open   int
	closed int
}

type stringSet map[string]struct{}

func newStringSet() stringSet {
	return make(stringSet)
}

func (s stringSet) add(value string) {
	s[value] = struct{}{}
}

func (s stringSet) has(value string) bool {
	_, ok := s[value]
	return ok
}

func sortedValues(values stringSet) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

type ownerGroups struct {
	users    stringSet
	teams    stringSet
	partial  stringSet
	excluded stringSet
}

func logins(value string, stripLeadingSlash bool) stringSet {
	result := newStringSet()
	for _, token := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}) {
		token = strings.TrimPrefix(strings.ToLower(token), "@")
		if stripLeadingSlash {
			token = strings.TrimLeft(token, "/")
		}
		if token != "" {
			result.add(token)
		}
	}
	return result
}

func parseCodeowners(text string, excluded string) ownerGroups {
	groups := ownerGroups{
		users:    newStringSet(),
		teams:    newStringSet(),
		partial:  newStringSet(),
		excluded: newStringSet(),
	}
	counts := map[string]map[string]int{"user": {}, "team": {}}
	catchAll := newStringSet()
	rules := 0
	for line := range strings.Lines(text) {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		rules++
		seen := newStringSet()
		for _, field := range fields[1:] {
			if !strings.HasPrefix(field, "@") {
				continue
			}
			owner := strings.ToLower(strings.TrimPrefix(field, "@"))
			if seen.has(owner) {
				continue
			}
			seen.add(owner)
			if fields[0] == "*" {
				catchAll.add(owner)
			}
			kind := "user"
			if strings.Contains(owner, "/") {
				kind = "team"
			}
			counts[kind][owner]++
		}
	}

	excludedOwners := logins(excluded, true)
	for _, kind := range []string{"user", "team"} {
		for owner, count := range counts[kind] {
			switch {
			case excludedOwners.has(owner):
				groups.excluded.add(owner)
			case count == rules || catchAll.has(owner):
				if kind == "user" {
					groups.users.add(owner)
				} else {
					groups.teams.add(owner)
				}
			default:
				groups.partial.add(owner)
			}
		}
	}
	return groups
}

func readCodeowners(ctx context.Context, client *githubClient, repo string) (string, error) {
	var repository struct {
		DefaultBranch string `json:"default_branch"`
	}
	if _, err := client.request(ctx, http.MethodGet, "repos/"+repo, nil, nil, &repository); err != nil {
		return "", err
	}
	for _, path := range codeownersPaths {
		var content struct {
			Content string `json:"content"`
		}
		query := url.Values{"ref": {repository.DefaultBranch}}
		if _, err := client.request(ctx, http.MethodGet, "repos/"+repo+"/contents/"+path, query, nil, &content); err != nil {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			return "", fmt.Errorf("decode %s: %w", path, err)
		}
		fmt.Printf("Reading %s from %s.\n", path, repository.DefaultBranch)
		return string(decoded), nil
	}
	return "", errors.New("no CODEOWNERS file found at any known location")
}

func resolvePool(ctx context.Context, client *githubClient, cfg config, groups ownerGroups) ([]string, error) {
	assignees, err := paginate[user](ctx, client, "repos/"+cfg.repo+"/assignees", url.Values{"per_page": {"100"}}, false)
	if err != nil {
		return nil, err
	}
	assignable := newStringSet()
	for _, assignee := range assignees {
		assignable.add(strings.ToLower(assignee.Login))
	}

	pool := newStringSet()
	if cfg.assigneePool != "" {
		pool = logins(cfg.assigneePool, false)
		fmt.Println("Using ASSIGNEE_POOL for the code owner pool.")
	} else {
		for owner := range groups.users {
			pool.add(owner)
		}
		org, _, _ := strings.Cut(cfg.repo, "/")
		for _, team := range sortedValues(groups.teams) {
			_, slug, _ := strings.Cut(team, "/")
			members, err := paginate[user](ctx, client, "orgs/"+org+"/teams/"+slug+"/members", url.Values{"per_page": {"100"}}, false)
			if err != nil {
				fmt.Printf("::warning::Could not expand team @%s: listing org team members needs an org-scoped token. Set the CODEOWNER_ASSIGNEES repository variable to supply the pool directly.\n", team)
				continue
			}
			for _, member := range members {
				pool.add(strings.ToLower(member.Login))
			}
			fmt.Printf("Expanded @%s to %d member(s).\n", team, len(members))
		}
	}

	excluded := logins(cfg.excludedOwners, true)
	eligible := newStringSet()
	for owner := range pool {
		if !excluded.has(owner) && assignable.has(owner) {
			eligible.add(owner)
		}
	}
	return sortedValues(eligible), nil
}

type prOwner struct {
	number int
	owner  string
}

func calculateLoad(openPRs, closedPRs []pullRequest, eligible []string, now time.Time, maxOpenAgeDays int) (map[string]ownerLoad, stringSet) {
	load := make(map[string]ownerLoad, len(eligible))
	for _, owner := range eligible {
		load[owner] = ownerLoad{}
	}
	staleOwners := newStringSet()
	for _, state := range []struct {
		name  string
		pulls []pullRequest
	}{{"open", openPRs}, {"closed", closedPRs}} {
		seen := make(map[prOwner]struct{})
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
				if state.name == "open" {
					age := int(math.Floor(now.Sub(pr.CreatedAt).Hours() / 24))
					if age > maxOpenAgeDays {
						staleOwners.add(owner)
						continue
					}
				}
				counts, ok := load[owner]
				if !ok {
					continue
				}
				if state.name == "open" {
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

func findCandidates(pulls []pullRequest, eligible []string, markerLabel string) []candidate {
	owners := newStringSet()
	for _, owner := range eligible {
		owners.add(owner)
	}
	byNumber := make(map[int]candidate)
	for _, pr := range pulls {
		if len(pr.Assignees) > 0 || pr.Draft || pr.User == nil || pr.User.Type == "Bot" || owners.has(strings.ToLower(pr.User.Login)) {
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
		byNumber[pr.Number] = candidate{number: pr.Number, author: pr.User.Login, marked: marked}
	}
	result := make([]candidate, 0, len(byNumber))
	for _, candidate := range byNumber {
		result = append(result, candidate)
	}
	slices.SortFunc(result, func(a, b candidate) int { return a.number - b.number })
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
		}
	}
	return lastHuman > lastBot
}

func chooseAssignee(load map[string]ownerLoad, author string, choose func([]string) string) (string, bool) {
	owners := make([]string, 0, len(load))
	for owner := range load {
		owners = append(owners, owner)
	}
	slices.Sort(owners)
	author = strings.ToLower(author)
	var bestKey [2]int
	var choices []string
	for _, owner := range owners {
		if owner == author {
			continue
		}
		counts := load[owner]
		key := [2]int{counts.open + counts.closed, counts.open}
		if len(choices) == 0 || key[0] < bestKey[0] || (key[0] == bestKey[0] && key[1] < bestKey[1]) {
			bestKey = key
			choices = []string{owner}
		} else if key == bestKey {
			choices = append(choices, owner)
		}
	}
	if len(choices) == 0 {
		return "", false
	}
	return choose(choices), true
}

func ensureMarkerLabel(ctx context.Context, client *githubClient, cfg config) error {
	labelPath := url.PathEscape(cfg.markerLabel)
	body := map[string]string{
		"color":       "ededed",
		"description": "Assigned by the daily code owner sweep",
	}
	_, err := client.request(ctx, http.MethodPatch, "repos/"+cfg.repo+"/labels/"+labelPath, nil, body, nil)
	if err == nil {
		return nil
	}
	var statusError *apiError
	if !errors.As(err, &statusError) || statusError.status != http.StatusNotFound {
		return err
	}
	body["name"] = cfg.markerLabel
	_, err = client.request(ctx, http.MethodPost, "repos/"+cfg.repo+"/labels", nil, body, nil)
	return err
}

func randomChoice(choices []string) string {
	return choices[rand.IntN(len(choices))]
}

func runSweep(ctx context.Context, client *githubClient, cfg config, now time.Time, choose func([]string) string) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if choose == nil {
		choose = randomChoice
	}

	codeowners, err := readCodeowners(ctx, client, cfg.repo)
	if err != nil {
		return err
	}
	groups := parseCodeowners(codeowners, cfg.excludedOwners)
	if len(groups.excluded) > 0 {
		fmt.Printf("::notice::Excluded from the pool by EXCLUDED_OWNERS: %s\n", strings.Join(sortedValues(groups.excluded), " "))
	}
	if len(groups.partial) > 0 {
		fmt.Printf("::notice::These CODEOWNERS owners appear in only some rules and are excluded from the pool: %s\n", strings.Join(sortedValues(groups.partial), " "))
	}

	eligible, err := resolvePool(ctx, client, cfg, groups)
	if err != nil {
		return err
	}
	if len(eligible) == 0 {
		return fmt.Errorf("no assignable code owner was found. Pool: users [%s], teams [%s]; excluded as partial [%s]. Restore the hardcoded ASSIGNEE_POOL default, set the CODEOWNER_ASSIGNEES repository variable, or expand a team with an org-scoped token",
			strings.Join(sortedValues(groups.users), " "), strings.Join(sortedValues(groups.teams), " "), strings.Join(sortedValues(groups.partial), " "))
	}
	fmt.Printf("Eligible owners: %d.\n", len(eligible))

	openPRs, err := paginate[pullRequest](ctx, client, "repos/"+cfg.repo+"/pulls", url.Values{"state": {"open"}, "per_page": {"100"}}, false)
	if err != nil {
		return err
	}
	since := now.Add(-7 * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")
	closedQuery := fmt.Sprintf("repo:%s is:pr is:closed closed:>=%s", cfg.repo, since)
	var search struct {
		TotalCount int `json:"total_count"`
	}
	if _, err := client.request(ctx, http.MethodGet, "search/issues", url.Values{"q": {closedQuery}, "per_page": {"1"}}, nil, &search); err != nil {
		return err
	}
	if search.TotalCount > 1000 {
		fmt.Printf("::warning::%d PRs closed in the last seven days, but the search API returns at most 1000. Closed load is undercounted.\n", search.TotalCount)
	}
	closedPRs, err := paginate[pullRequest](ctx, client, "search/issues", url.Values{"q": {closedQuery}, "per_page": {"100"}}, true)
	if err != nil {
		return err
	}
	load, staleOwners := calculateLoad(openPRs, closedPRs, eligible, now, cfg.openLoadMaxAgeDays)
	for _, owner := range sortedValues(staleOwners) {
		fmt.Fprintf(os.Stderr, "Open PRs by %s older than %d days no longer count as load.\n", owner, cfg.openLoadMaxAgeDays)
	}

	// Fetch again because assignments may have changed while load was calculated.
	currentPRs, err := paginate[pullRequest](ctx, client, "repos/"+cfg.repo+"/pulls", url.Values{"state": {"open"}, "per_page": {"100"}}, false)
	if err != nil {
		return err
	}
	candidates := findCandidates(currentPRs, eligible, cfg.markerLabel)
	fmt.Printf("Found %d unassigned PR(s).\n", len(candidates))
	if len(candidates) == 0 {
		return nil
	}
	if err := ensureMarkerLabel(ctx, client, cfg); err != nil {
		fmt.Printf("::warning::Could not create the %s label.\n", cfg.markerLabel)
	}

	assigned, skipped, failed, unassignable := 0, 0, 0, 0
	for _, candidate := range candidates {
		if assigned >= cfg.maxAssignments {
			fmt.Printf("Reached the per-run cap of %d; the rest are left for the next run.\n", cfg.maxAssignments)
			break
		}
		if candidate.marked {
			events, err := paginate[timelineEvent](ctx, client, fmt.Sprintf("repos/%s/issues/%d/timeline", cfg.repo, candidate.number), url.Values{"per_page": {"100"}}, false)
			if err != nil {
				fmt.Printf("::warning::Could not read the timeline for PR #%d; leaving it for the next run.\n", candidate.number)
				failed++
				continue
			}
			if deliberatelyUnassigned(events, cfg.botActor) {
				fmt.Printf("PR #%d was auto-assigned before and unassigned by a person; skipping.\n", candidate.number)
				skipped++
				continue
			}
		}

		assignee, ok := chooseAssignee(load, candidate.author, choose)
		if !ok {
			fmt.Printf("::warning::No eligible owner other than @%s for PR #%d.\n", candidate.author, candidate.number)
			unassignable++
			continue
		}
		var current struct {
			Assignees []user `json:"assignees"`
		}
		if _, err := client.request(ctx, http.MethodGet, fmt.Sprintf("repos/%s/issues/%d", cfg.repo, candidate.number), nil, nil, &current); err != nil {
			fmt.Printf("::warning::Could not re-read PR #%d before assigning.\n", candidate.number)
			failed++
			continue
		}
		if len(current.Assignees) > 0 {
			fmt.Printf("PR #%d was assigned elsewhere while the sweep ran; leaving it alone.\n", candidate.number)
			skipped++
			continue
		}

		var response pullRequest
		if _, err := client.request(ctx, http.MethodPost, fmt.Sprintf("repos/%s/issues/%d/assignees", cfg.repo, candidate.number), nil, map[string][]string{"assignees": {assignee}}, &response); err != nil {
			fmt.Printf("::warning::Could not assign PR #%d to @%s.\n", candidate.number, assignee)
			failed++
			continue
		}
		reflected := false
		for _, returned := range response.Assignees {
			if strings.EqualFold(returned.Login, assignee) {
				reflected = true
				break
			}
		}
		if !reflected {
			fmt.Printf("::warning::PR #%d assignment to @%s was not reflected in the response.\n", candidate.number, assignee)
			failed++
			continue
		}
		if _, err := client.request(ctx, http.MethodPost, fmt.Sprintf("repos/%s/issues/%d/labels", cfg.repo, candidate.number), nil, map[string][]string{"labels": {cfg.markerLabel}}, nil); err != nil {
			fmt.Printf("::warning::Could not mark PR #%d with %s.\n", candidate.number, cfg.markerLabel)
		}

		counts := load[assignee]
		counts.open++
		load[assignee] = counts
		fmt.Printf("Assigned @%s to PR #%d (load: %d = %d open + %d closed in the past seven days).\n", assignee, candidate.number, counts.open+counts.closed, counts.open, counts.closed)
		assigned++
	}

	fmt.Printf("Done: assigned %d, skipped %d deliberately unassigned, %d with no eligible owner, %d deferred by errors.\n", assigned, skipped, unassignable, failed)
	if failed > 0 {
		return fmt.Errorf("%d PR(s) could not be processed; they will be retried on the next run", failed)
	}
	return nil
}

func reportFailure(ctx context.Context, client *githubClient, repo, runURL string) error {
	title := "assign-codeowner sweep failed"
	body := fmt.Sprintf("The daily code owner sweep failed.\n\nRun: %s\n\nThis is expected when the code owner pool resolves to nobody: CODEOWNERS may name teams only, and a team cannot be assigned to a pull request. Set the CODEOWNER_ASSIGNEES repository variable, or name individuals in CODEOWNERS.\n", runURL)
	var existing *int
	var results struct {
		Items []issue `json:"items"`
	}
	query := url.Values{"q": {fmt.Sprintf(`repo:%s is:issue is:open in:title "%s"`, repo, title)}}
	if _, err := client.request(ctx, http.MethodGet, "search/issues", query, nil, &results); err == nil && len(results.Items) > 0 {
		existing = &results.Items[0].Number
	}
	if existing != nil {
		_, err := client.request(ctx, http.MethodPost, fmt.Sprintf("repos/%s/issues/%d/comments", repo, *existing), nil, map[string]string{"body": body}, nil)
		return err
	}
	_, err := client.request(ctx, http.MethodPost, "repos/"+repo+"/issues", nil, map[string]string{"title": title, "body": body}, nil)
	return err
}

func run(ctx context.Context, command string) error {
	if command != "sweep" && command != "report-failure" {
		return fmt.Errorf("unknown command %q; expected sweep or report-failure", command)
	}
	cfg, err := configFromEnv()
	if err != nil {
		return err
	}
	token, err := requiredEnv("GH_TOKEN")
	if err != nil {
		return err
	}
	client := newGitHubClient(token)
	if command == "report-failure" {
		runURL, err := requiredEnv("RUN_URL")
		if err != nil {
			return err
		}
		return reportFailure(ctx, client, cfg.repo, runURL)
	}
	return runSweep(ctx, client, cfg, time.Time{}, nil)
}

func workflowError(err error) {
	message := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(err.Error())
	fmt.Printf("::error::%s\n", message)
}

func main() {
	command := "sweep"
	if len(os.Args) > 2 {
		workflowError(errors.New("usage: assign_codeowner [sweep|report-failure]"))
		os.Exit(2)
	}
	if len(os.Args) == 2 {
		command = os.Args[1]
	}
	if err := run(context.Background(), command); err != nil {
		workflowError(err)
		os.Exit(1)
	}
}
