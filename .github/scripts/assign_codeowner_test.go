package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testNow = time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)

func firstChoice(choices []string) string {
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
	require.Equal(t, []string{"alice"}, sortedValues(groups.users))
	require.Equal(t, []string{"grafana/tempo"}, sortedValues(groups.teams))
	require.Equal(t, []string{"bob"}, sortedValues(groups.partial))
	require.Equal(t, []string{"grafana/docs-traces"}, sortedValues(groups.excluded))
}

func TestLoadDeduplicatesAndExpiresStaleOpenPRs(t *testing.T) {
	stale := assignedTo(testPull(2), "Alice")
	stale.CreatedAt = testNow.Add(-91 * 24 * time.Hour)
	bot := assignedTo(testPull(4), "Bob")
	bot.User = &user{Login: "dependabot[bot]", Type: "Bot"}
	openPRs := []pullRequest{
		assignedTo(testPull(1), "Alice", "Alice"),
		stale,
		bot,
	}
	closedPRs := []pullRequest{assignedTo(testPull(4), "Alice", "Alice")}
	load, staleOwners := calculateLoad(openPRs, closedPRs, []string{"alice", "bob"}, testNow, 90)
	require.Equal(t, ownerLoad{open: 1, closed: 1}, load["alice"])
	require.Equal(t, ownerLoad{}, load["bob"])
	require.Equal(t, []string{"alice"}, sortedValues(staleOwners))
}

func TestLoadUsesBotTypeNotAuthorName(t *testing.T) {
	pr := assignedTo(testPull(1), "Alice")
	pr.User = &user{Login: "renovate[bot]", Type: "Bot"}
	load, _ := calculateLoad([]pullRequest{pr}, nil, []string{"alice"}, testNow, 90)
	require.Equal(t, ownerLoad{}, load["alice"])
}

func TestCandidatesFilterAndOrderOldestFirst(t *testing.T) {
	marked := testPull(2)
	marked.Labels = []label{{Name: "codeowner-sweep"}}
	bot := testPull(6)
	bot.User = &user{Login: "dependabot[bot]", Type: "Bot"}
	deletedAuthor := testPull(4)
	deletedAuthor.User = nil
	draft := testPull(3)
	draft.Draft = true
	ownerAuthored := testPull(7)
	ownerAuthored.User.Login = "Alice"
	assigned := assignedTo(testPull(5), "Bob")
	candidates := findCandidates([]pullRequest{ownerAuthored, assigned, marked, testPull(1), bot, deletedAuthor, draft}, []string{"alice", "bob"}, "codeowner-sweep")
	require.Equal(t, []candidate{{number: 1, author: "external"}, {number: 2, author: "external", marked: true}}, candidates)
}

func TestAssigneeTieBreakPrefersLessOpenLoadThenRandomizes(t *testing.T) {
	load := map[string]ownerLoad{
		"alice": {open: 1, closed: 1},
		"bob":   {open: 0, closed: 2},
		"carol": {open: 3},
	}
	chosen, ok := chooseAssignee(load, "external", firstChoice)
	require.True(t, ok)
	require.Equal(t, "bob", chosen)
	load["bob"] = ownerLoad{open: 1, closed: 2}
	chosen, ok = chooseAssignee(load, "external", firstChoice)
	require.True(t, ok)
	require.Equal(t, "alice", chosen)
	_, ok = chooseAssignee(map[string]ownerLoad{"alice": load["alice"]}, "Alice", firstChoice)
	require.False(t, ok)
}

func TestOnlyHumanUnassignAfterLatestBotAssignmentIsSkipped(t *testing.T) {
	const bot = "github-actions[bot]"
	assigned := timelineEvent{Event: "assigned", Actor: &user{Login: bot}}
	humanUnassigned := timelineEvent{Event: "unassigned", Actor: &user{Login: "maintainer"}}
	automaticUnassigned := timelineEvent{Event: "unassigned"}
	require.True(t, deliberatelyUnassigned([]timelineEvent{assigned, humanUnassigned}, bot))
	require.False(t, deliberatelyUnassigned([]timelineEvent{humanUnassigned, assigned}, bot))
	require.False(t, deliberatelyUnassigned([]timelineEvent{assigned, automaticUnassigned}, bot))
}

type createdIssue struct {
	Title string
	Body  string
}

type issueComment struct {
	Number int
	Body   string
}

type gitHubSimulation struct {
	repo             string
	pulls            map[int]*pullRequest
	labels           map[string]map[string]string
	createdIssues    []createdIssue
	comments         []issueComment
	failureIssue     int
	failRepoRequests int
	baseURL          string
}

func newGitHubSimulation() *gitHubSimulation {
	pr10, pr20 := testPull(10), testPull(20)
	return &gitHubSimulation{
		repo:             "grafana/tempo",
		pulls:            map[int]*pullRequest{20: &pr20, 10: &pr10},
		labels:           make(map[string]map[string]string),
		failRepoRequests: 1,
	}
}

var (
	issuePath   = regexp.MustCompile(`^/repos/grafana/tempo/issues/(\d+)$`)
	assignPath  = regexp.MustCompile(`^/repos/grafana/tempo/issues/(\d+)/assignees$`)
	labelPath   = regexp.MustCompile(`^/repos/grafana/tempo/issues/(\d+)/labels$`)
	commentPath = regexp.MustCompile(`^/repos/grafana/tempo/issues/(\d+)/comments$`)
)

func (s *gitHubSimulation) reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *gitHubSimulation) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	repoPath := "/repos/" + s.repo
	path := r.URL.Path
	if path == repoPath {
		if s.failRepoRequests > 0 {
			s.failRepoRequests--
			s.reply(w, http.StatusInternalServerError, map[string]string{"message": "temporary failure"})
		} else {
			s.reply(w, http.StatusOK, map[string]string{"default_branch": "main"})
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r, repoPath, path)
	case http.MethodPatch:
		s.handlePatch(w, r, repoPath, path)
	case http.MethodPost:
		s.handlePost(w, r, repoPath, path)
	default:
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown method"})
	}
}

func (s *gitHubSimulation) handleGet(w http.ResponseWriter, r *http.Request, repoPath, path string) {
	switch {
	case path == repoPath+"/contents/CODEOWNERS":
		content := base64.StdEncoding.EncodeToString([]byte("* @grafana/tempo\n"))
		s.reply(w, http.StatusOK, map[string]string{"content": content})
	case path == repoPath+"/assignees":
		s.reply(w, http.StatusOK, []user{{Login: "Alice"}, {Login: "Bob"}})
	case path == repoPath+"/pulls":
		if r.URL.Query().Get("page") == "2" {
			s.reply(w, http.StatusOK, []pullRequest{*s.pulls[10]})
			return
		}
		next := fmt.Sprintf("<%s%s?state=open&per_page=100&page=2>; rel=\"next\"", s.baseURL, path)
		w.Header().Set("Link", next)
		s.reply(w, http.StatusOK, []pullRequest{*s.pulls[20]})
	case path == "/search/issues":
		if strings.Contains(r.URL.Query().Get("q"), "is:issue") {
			items := []issue{}
			if s.failureIssue != 0 {
				items = append(items, issue{Number: s.failureIssue})
			}
			s.reply(w, http.StatusOK, map[string]any{"total_count": len(items), "items": items})
			return
		}
		s.reply(w, http.StatusOK, map[string]any{"total_count": 0, "items": []pullRequest{}})
	case strings.HasSuffix(path, "/timeline"):
		s.reply(w, http.StatusOK, []timelineEvent{})
	default:
		if match := issuePath.FindStringSubmatch(path); match != nil {
			var number int
			_, _ = fmt.Sscanf(match[1], "%d", &number)
			s.reply(w, http.StatusOK, s.pulls[number])
			return
		}
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown GET " + path})
	}
}

func (s *gitHubSimulation) handlePatch(w http.ResponseWriter, r *http.Request, repoPath, path string) {
	if path != repoPath+"/labels/codeowner-sweep" {
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown PATCH " + path})
		return
	}
	name := strings.TrimPrefix(path, repoPath+"/labels/")
	if _, ok := s.labels[name]; !ok {
		s.reply(w, http.StatusNotFound, map[string]string{"message": "not found"})
		return
	}
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	s.labels[name] = body
	s.reply(w, http.StatusOK, body)
}

func (s *gitHubSimulation) handlePost(w http.ResponseWriter, r *http.Request, repoPath, path string) {
	var body struct {
		Name      string   `json:"name"`
		Color     string   `json:"color"`
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Assignees []string `json:"assignees"`
		Labels    []string `json:"labels"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch {
	case path == repoPath+"/labels":
		s.labels[body.Name] = map[string]string{"name": body.Name, "color": body.Color}
		s.reply(w, http.StatusCreated, body)
	case assignPath.MatchString(path):
		match := assignPath.FindStringSubmatch(path)
		var number int
		_, _ = fmt.Sscanf(match[1], "%d", &number)
		canonical := map[string]string{"alice": "Alice", "bob": "Bob"}
		for _, login := range body.Assignees {
			s.pulls[number].Assignees = append(s.pulls[number].Assignees, user{Login: canonical[login]})
		}
		s.reply(w, http.StatusOK, s.pulls[number])
	case labelPath.MatchString(path):
		match := labelPath.FindStringSubmatch(path)
		var number int
		_, _ = fmt.Sscanf(match[1], "%d", &number)
		for _, name := range body.Labels {
			s.pulls[number].Labels = append(s.pulls[number].Labels, label{Name: name})
		}
		s.reply(w, http.StatusOK, s.pulls[number].Labels)
	case path == repoPath+"/issues":
		s.createdIssues = append(s.createdIssues, createdIssue{Title: body.Title, Body: body.Body})
		s.reply(w, http.StatusCreated, issue{Number: 100 + len(s.createdIssues)})
	case commentPath.MatchString(path):
		match := commentPath.FindStringSubmatch(path)
		var number int
		_, _ = fmt.Sscanf(match[1], "%d", &number)
		s.comments = append(s.comments, issueComment{Number: number, Body: body.Body})
		s.reply(w, http.StatusCreated, issue{Number: len(s.comments)})
	default:
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown POST " + path})
	}
}

func newSimulationClient(t *testing.T) (*gitHubSimulation, *githubClient) {
	t.Helper()
	simulation := newGitHubSimulation()
	server := httptest.NewServer(simulation)
	t.Cleanup(server.Close)
	simulation.baseURL = server.URL
	client := newGitHubClient("test-token")
	client.baseURL = server.URL
	client.sleep = func(context.Context, time.Duration) error { return nil }
	return simulation, client
}

func testConfig() config {
	return config{
		repo:               "grafana/tempo",
		assigneePool:       "Alice,Bob",
		maxAssignments:     1,
		openLoadMaxAgeDays: 90,
		markerLabel:        "codeowner-sweep",
		botActor:           "github-actions[bot]",
	}
}

func TestSweepRetriesPaginatesAndAssignsOldestUpToCap(t *testing.T) {
	simulation, client := newSimulationClient(t)
	require.NoError(t, runSweep(context.Background(), client, testConfig(), testNow, firstChoice))
	require.Equal(t, 0, simulation.failRepoRequests)
	require.Equal(t, []user{{Login: "Alice"}}, simulation.pulls[10].Assignees)
	require.Empty(t, simulation.pulls[20].Assignees)
	require.Equal(t, []label{{Name: "codeowner-sweep"}}, simulation.pulls[10].Labels)
	require.Contains(t, simulation.labels, "codeowner-sweep")
}

func TestSweepSpreadsBatchAcrossTiedOwners(t *testing.T) {
	simulation, client := newSimulationClient(t)
	cfg := testConfig()
	cfg.maxAssignments = 2
	require.NoError(t, runSweep(context.Background(), client, cfg, testNow, firstChoice))
	require.Equal(t, []user{{Login: "Alice"}}, simulation.pulls[10].Assignees)
	require.Equal(t, []user{{Login: "Bob"}}, simulation.pulls[20].Assignees)
}

func TestFailureReportCreatesThenReusesIssue(t *testing.T) {
	simulation, client := newSimulationClient(t)
	const runURL = "https://github.com/grafana/tempo/actions/runs/123"
	require.NoError(t, reportFailure(context.Background(), client, simulation.repo, runURL))
	require.Len(t, simulation.createdIssues, 1)
	require.Contains(t, simulation.createdIssues[0].Body, runURL)

	simulation.failureIssue = 42
	require.NoError(t, reportFailure(context.Background(), client, simulation.repo, runURL))
	require.Len(t, simulation.createdIssues, 1)
	require.Equal(t, []issueComment{{Number: 42, Body: simulation.createdIssues[0].Body}}, simulation.comments)
}

func TestNextPage(t *testing.T) {
	require.Equal(t, "https://api.github.com/page=2", nextPage(`<https://api.github.com/page=2>; rel="next", <https://api.github.com/page=4>; rel="last"`))
	require.Empty(t, nextPage(`<https://api.github.com/page=4>; rel="last"`))
	parsed, err := url.Parse(nextPage(`<https://api.github.com/page=2>; rel="next"`))
	require.NoError(t, err)
	require.Equal(t, "api.github.com", parsed.Host)
}
