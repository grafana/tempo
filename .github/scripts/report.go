package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/v93/github"
)

// search query is a substring match on the title; search indexing lag can
// briefly produce two issues for consecutive failures, and nothing closes
// stale ones. Results are sorted oldest-first so the recipient issue is stable.
func reportFailure(ctx context.Context, client *github.Client, runURL string) error {
	title := failureIssueTitle
	body := fmt.Sprintf(`The daily code owner sweep failed.

Run: %s

The most common cause is a stale member in the hardcoded pool
(.github/scripts/consts.go): assigning someone who is no longer assignable
in the repository returns a 422, so remove them from defaultAssigneePool in
the same change that removed their access. Transient API errors are retried
and self-heal on the next run; inspect the run log for the exact failure.
`, runURL)
	query := url.Values{"q": {fmt.Sprintf(`repo:%s is:issue is:open in:title "%s"`, defaultRepo, title)}}
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
		if _, _, err := client.Issues.CreateComment(ctx, repoOwner, repoName, number, github.IssueCommentRequest{Body: body}); err != nil {
			return fmt.Errorf("comment on sweep failure issue #%d: %w", number, err)
		}
		return nil
	}
	if _, _, err := client.Issues.Create(ctx, repoOwner, repoName, github.CreateIssueRequest{Title: title, Body: &body}); err != nil {
		return fmt.Errorf("create sweep failure issue: %w", err)
	}
	return nil
}
