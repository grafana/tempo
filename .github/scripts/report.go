package main

import (
	"context"
	"fmt"
	"net/http"
)

func reportFailure(ctx context.Context, client *githubClient, repo, runURL string) error {
	title := "assign-codeowner sweep failed"
	body := fmt.Sprintf("The daily code owner sweep failed.\n\nRun: %s\n\nThis is expected when the code owner pool resolves to nobody: CODEOWNERS may name teams only, and a team cannot be assigned to a pull request. Set the CODEOWNER_ASSIGNEES repository variable, or name individuals in CODEOWNERS.\n", runURL)
	var results struct {
		Items []issue `json:"items"`
	}
	query := queryWith("q", fmt.Sprintf(`repo:%s is:issue is:open in:title "%s"`, repo, title))
	if _, err := client.request(ctx, http.MethodGet, "search/issues", query, nil, &results); err != nil {
		return fmt.Errorf("find existing sweep failure issue: %w", err)
	}
	if len(results.Items) > 0 {
		number := results.Items[0].Number
		if _, err := client.request(ctx, http.MethodPost, issueEndpoint(repo, number, "/comments"), nil, map[string]string{"body": body}, nil); err != nil {
			return fmt.Errorf("comment on sweep failure issue #%d: %w", number, err)
		}
		return nil
	}
	if _, err := client.request(ctx, http.MethodPost, repositoryEndpoint(repo, "/issues"), nil, map[string]string{"title": title, "body": body}, nil); err != nil {
		return fmt.Errorf("create sweep failure issue: %w", err)
	}
	return nil
}
