package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v93/github"
)

// defaultAssigneePool is the hardcoded code owner pool for the scheduled
// sweep. Tempo's CODEOWNERS names only the @grafana/tempo team, and team
// expansion needs an org-scoped token this workflow does not have, so the pool
// is the global "*" owners from before #8133 switched CODEOWNERS to the team.
// Entries must be lowercase GitHub logins without a leading @; resolvePool
// matches them case-insensitively against the repository's assignable users.
// Change this list in the same commit as the corresponding CODEOWNERS change.
var defaultAssigneePool = []string{
	"mdisibio",
	"mapno",
	"zalegrala",
	"electron0zero",
	"ie-pham",
	"stoewer",
	"javiermolinar",
	"carles-grafana",
	"ruslan-mikhailov",
	"mattdurham",
	"zhxiaogg",
}

// resolvePool derives the eligible assignees: the hardcoded pool intersected
// with the repository's assignable users, so a stale pool entry cannot trigger
// a 422 assignment storm. Pool order is preserved, which keeps tie-breaking
// deterministic.
func resolvePool(ctx context.Context, client *github.Client) ([]string, error) {
	owner, name, err := splitRepo(defaultRepo)
	if err != nil {
		return nil, err
	}
	assignees, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.User, *github.Response, error) {
		return client.Issues.ListAssignees(ctx, owner, name, opts)
	})
	if err != nil {
		return nil, fmt.Errorf("list repository assignees: %w", err)
	}
	assignable := make(map[string]struct{}, len(assignees))
	for _, assignee := range assignees {
		assignable[strings.ToLower(assignee.GetLogin())] = struct{}{}
	}

	eligible := make([]string, 0, len(defaultAssigneePool))
	for _, login := range defaultAssigneePool {
		if _, ok := assignable[login]; ok {
			eligible = append(eligible, login)
		}
	}
	return eligible, nil
}
