package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/v93/github"
)

// failureIssueTitle identifies failure issues in both the dedupe search and
// the created issue; the two must stay in lockstep.
const failureIssueTitle = "assign-codeowner sweep failed"

// reportFailure records a failed sweep run where maintainers will see it. The
// search query is a substring match on the title; search indexing lag can
// briefly produce two issues for consecutive failures, and nothing closes
// stale ones. Results are sorted oldest-first so the recipient issue is stable.
func reportFailure(ctx context.Context, client *github.Client, repo, runURL string) error {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return err
	}
	title := failureIssueTitle
	body := fmt.Sprintf(`The daily code owner sweep failed.

Run: %s

The most common cause is the code owner pool resolving to nobody: the pool
and exclusions are hardcoded in .github/scripts (config.go and pool.go), and
every pool member must be assignable in the repository. Inspect the run log
for the exact failure.
`, runURL)
	query := url.Values{"q": {fmt.Sprintf(`repo:%s is:issue is:open in:title "%s"`, repo, title)}}
	var result *github.IssuesSearchResult
	if err := retryableCall(ctx, func() error {
		var callErr error
		result, _, callErr = client.Search.Issues(ctx, query.Get("q"), &github.SearchOptions{Sort: "created", Order: "asc"})
		return callErr
	}); err != nil {
		return fmt.Errorf("find existing sweep failure issue: %w", err)
	}
	if len(result.Issues) > 0 {
		number := result.Issues[0].GetNumber()
		if _, _, err := client.Issues.CreateComment(ctx, owner, name, number, github.IssueCommentRequest{Body: body}); err != nil {
			return fmt.Errorf("comment on sweep failure issue #%d: %w", number, err)
		}
		return nil
	}
	if _, _, err := client.Issues.Create(ctx, owner, name, github.CreateIssueRequest{Title: title, Body: &body}); err != nil {
		return fmt.Errorf("create sweep failure issue: %w", err)
	}
	return nil
}
