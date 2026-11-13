package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type createdIssue struct {
	Title string
	Body  string
}

type issueComment struct {
	Number int
	Body   string
}

type simulationRequestBody struct {
	Name      string   `json:"name"`
	Color     string   `json:"color"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Assignees []string `json:"assignees"`
	Labels    []string `json:"labels"`
}

type gitHubSimulation struct {
	mu                 sync.Mutex
	repo               string
	pulls              map[int]*pullRequest
	labels             map[string]map[string]string
	createdIssues      []createdIssue
	comments           []issueComment
	failureIssue       int
	failRepoRequests   int
	failMarkerApply    bool
	failIssueSearch    bool
	concurrentAssignee string
	baseURL            string
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

func (s *gitHubSimulation) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repoPath := "/repos/" + s.repo
	if r.URL.Path == repoPath {
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
		s.handleGet(w, r, repoPath)
	case http.MethodPatch:
		s.handlePatch(w, r, repoPath)
	case http.MethodPost:
		s.handlePost(w, r, repoPath)
	case http.MethodDelete:
		s.handleDelete(w, r, repoPath)
	default:
		// Unsupported methods indicate a fixture/API-contract mismatch.
		s.reply(w, http.StatusMethodNotAllowed, map[string]string{"message": "unsupported method " + r.Method})
	}
}

func (s *gitHubSimulation) handleGet(w http.ResponseWriter, r *http.Request, repoPath string) {
	path := r.URL.Path
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
			if s.failIssueSearch {
				s.reply(w, http.StatusInternalServerError, map[string]string{"message": "search unavailable"})
				return
			}
			items := []issue{}
			if s.failureIssue != 0 {
				items = append(items, issue{Number: s.failureIssue})
			}
			s.reply(w, http.StatusOK, map[string]any{"total_count": len(items), "items": items})
			return
		}
		s.reply(w, http.StatusOK, map[string]any{"total_count": 0, "items": []pullRequest{}})
	default:
		if number, ok := parseIssuePath(path, s.repo, "timeline"); ok {
			_ = number // Empty default timelines are sufficient for these tests.
			s.reply(w, http.StatusOK, []timelineEvent{})
			return
		}
		if number, ok := parseIssuePath(path, s.repo, ""); ok {
			if pr := s.pulls[number]; pr != nil {
				s.reply(w, http.StatusOK, pr)
			} else {
				s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown PR"})
			}
			return
		}
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown GET " + path})
	}
}

func (s *gitHubSimulation) handlePatch(w http.ResponseWriter, r *http.Request, repoPath string) {
	prefix := repoPath + "/labels/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown PATCH " + r.URL.Path})
		return
	}
	name := strings.TrimPrefix(r.URL.Path, prefix)
	if _, ok := s.labels[name]; !ok {
		s.reply(w, http.StatusNotFound, map[string]string{"message": "not found"})
		return
	}
	var body map[string]string
	if !s.decode(w, r, &body) {
		return
	}
	s.labels[name] = body
	s.reply(w, http.StatusOK, body)
}

func (s *gitHubSimulation) handlePost(w http.ResponseWriter, r *http.Request, repoPath string) {
	var body simulationRequestBody
	if !s.decode(w, r, &body) {
		return
	}
	path := r.URL.Path
	switch {
	case path == repoPath+"/labels":
		s.labels[body.Name] = map[string]string{"name": body.Name, "color": body.Color}
		s.reply(w, http.StatusCreated, body)
	case path == repoPath+"/issues":
		s.createdIssues = append(s.createdIssues, createdIssue{Title: body.Title, Body: body.Body})
		s.reply(w, http.StatusCreated, issue{Number: 100 + len(s.createdIssues)})
	default:
		if number, ok := parseIssuePath(path, s.repo, "assignees"); ok {
			pr := s.pulls[number]
			if pr == nil {
				s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown PR"})
				return
			}
			if s.concurrentAssignee != "" {
				pr.Assignees = append(pr.Assignees, user{Login: s.concurrentAssignee})
			}
			canonical := map[string]string{"alice": "Alice", "bob": "Bob"}
			for _, login := range body.Assignees {
				pr.Assignees = append(pr.Assignees, user{Login: canonical[login]})
			}
			s.reply(w, http.StatusOK, pr)
			return
		}
		if number, ok := parseIssuePath(path, s.repo, "labels"); ok {
			pr := s.pulls[number]
			if pr == nil {
				s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown PR"})
				return
			}
			if s.failMarkerApply {
				s.reply(w, http.StatusInternalServerError, map[string]string{"message": "label write failed"})
				return
			}
			for _, name := range body.Labels {
				pr.Labels = append(pr.Labels, label{Name: name})
			}
			s.reply(w, http.StatusOK, pr.Labels)
			return
		}
		if number, ok := parseIssuePath(path, s.repo, "comments"); ok {
			s.comments = append(s.comments, issueComment{Number: number, Body: body.Body})
			s.reply(w, http.StatusCreated, issue{Number: len(s.comments)})
			return
		}
		// Unmatched POST paths indicate a new endpoint the simulator must implement.
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown POST " + path})
	}
}

func (s *gitHubSimulation) handleDelete(w http.ResponseWriter, r *http.Request, repoPath string) {
	number, ok := parseIssuePath(r.URL.Path, s.repo, "assignees")
	if !ok || s.pulls[number] == nil {
		s.reply(w, http.StatusNotFound, map[string]string{"message": "unknown assignee rollback"})
		return
	}
	var body simulationRequestBody
	if !s.decode(w, r, &body) {
		return
	}
	pr := s.pulls[number]
	for _, remove := range body.Assignees {
		kept := pr.Assignees[:0]
		for _, existing := range pr.Assignees {
			if !strings.EqualFold(existing.Login, remove) {
				kept = append(kept, existing)
			}
		}
		pr.Assignees = kept
	}
	s.reply(w, http.StatusOK, pr)
}

func parseIssuePath(path, repo, action string) (int, bool) {
	prefix := "/repos/" + repo + "/issues/"
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if action == "" && len(parts) != 1 || action != "" && (len(parts) != 2 || parts[1] != action) {
		return 0, false
	}
	number, err := strconv.Atoi(parts[0])
	return number, err == nil
}

func (s *gitHubSimulation) reply(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "simulation JSON encode failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		fmt.Printf("simulation response write failed: %v\n", err)
	}
}

func (s *gitHubSimulation) decode(w http.ResponseWriter, r *http.Request, out any) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		s.reply(w, http.StatusBadRequest, map[string]string{"message": "invalid simulation JSON: " + err.Error()})
		return false
	}
	return true
}

func newSimulationClient(t *testing.T) (*gitHubSimulation, *githubClient) {
	t.Helper()
	simulation := newGitHubSimulation()
	server := httptest.NewUnstartedServer(simulation)
	t.Cleanup(server.Close)
	simulation.baseURL = server.URL
	server.Start()
	client := newGitHubClient("test-token")
	client.baseURL = server.URL
	client.retryDelay = func(int) time.Duration { return 0 }
	return simulation, client
}

func testConfig() config {
	cfg := defaultConfig("grafana/tempo")
	cfg.assigneePool = "Alice,Bob"
	cfg.maxAssignments = 1
	return cfg
}

func (s *gitHubSimulation) pullSnapshot(number int) pullRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	pr := *s.pulls[number]
	pr.Assignees = append([]user(nil), pr.Assignees...)
	pr.Labels = append([]label(nil), pr.Labels...)
	return pr
}

func (s *gitHubSimulation) remainingRepoFailures() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failRepoRequests
}

func (s *gitHubSimulation) hasLabel(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.labels[name]
	return ok
}

func (s *gitHubSimulation) issueSnapshot() []createdIssue {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]createdIssue(nil), s.createdIssues...)
}

func (s *gitHubSimulation) commentSnapshot() []issueComment {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]issueComment(nil), s.comments...)
}

func (s *gitHubSimulation) setFailureIssue(number int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureIssue = number
}

func (s *gitHubSimulation) setFlag(flag *bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	*flag = true
}

func (s *gitHubSimulation) setConcurrentAssignee(login string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.concurrentAssignee = login
}

func runTestSweep(client *githubClient, cfg config) error {
	return runSweep(context.Background(), client, cfg, testNow, firstChoice)
}
