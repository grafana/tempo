package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/v93/github"
)

// queryWith builds a url.Values from key/value pairs. It panics on an odd
// argument count; callers must pass complete key/value pairs.
func queryWith(keyValues ...string) url.Values {
	if len(keyValues)%2 != 0 {
		panic("queryWith requires key/value pairs")
	}
	values := make(url.Values, len(keyValues)/2)
	for index := 0; index < len(keyValues); index += 2 {
		values.Set(keyValues[index], keyValues[index+1])
	}
	return values
}

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
	body := fmt.Sprintf("The daily code owner sweep failed.\n\nRun: %s\n\nThis is expected when the code owner pool resolves to nobody: CODEOWNERS may name teams only, and a team cannot be assigned to a pull request. Set the ASSIGNEE_POOL repository variable, or name individuals in CODEOWNERS.\n", runURL)
	query := queryWith("q", fmt.Sprintf(`repo:%s is:issue is:open in:title "%s"`, repo, title))
	result, _, err := client.Search.Issues(ctx, query.Get("q"), &github.SearchOptions{Sort: "created", Order: "asc"})
	if err != nil {
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
