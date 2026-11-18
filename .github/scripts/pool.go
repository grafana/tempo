package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/go-github/v93/github"
)

// defaultAssigneePool is the hardcoded code owner pool for the scheduled
// sweep. Tempo's CODEOWNERS names only the @grafana/tempo team, and team
// expansion needs an org-scoped token this workflow does not have, so the pool
// is the global "*" owners from before #8133 switched CODEOWNERS to the team.
// Change this list in the same commit as the corresponding CODEOWNERS change.
var defaultAssigneePool = []string{
	"mdisibio",
	"mapno",
	"yvrhdn",
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

// loginSet lowercases logins and drops a leading "@", so entries stay readable
// in the hardcoded lists above.
func loginSet(logins []string) map[string]struct{} {
	result := make(map[string]struct{}, len(logins))
	for _, login := range logins {
		login = strings.TrimPrefix(strings.ToLower(login), "@")
		if login != "" {
			result[login] = struct{}{}
		}
	}
	return result
}

func hasLogin(set map[string]struct{}, login string) bool {
	_, ok := set[login]
	return ok
}

// resolvePool derives the eligible assignees: the configured pool minus the
// excluded set, intersected with the repository's assignable users so a stale
// pool entry cannot trigger a 422 assignment storm.
func resolvePool(ctx context.Context, client *github.Client, cfg config) ([]string, error) {
	owner, name, err := splitRepo(cfg.repo)
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

	pool := loginSet(cfg.assigneePool)
	excluded := loginSet(cfg.excludedOwners)
	eligible := make(map[string]struct{}, len(pool))
	for login := range pool {
		if !hasLogin(excluded, login) && hasLogin(assignable, login) {
			eligible[login] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(eligible)), nil
}
