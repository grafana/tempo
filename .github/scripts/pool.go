package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/google/go-github/v93/github"
)

// defaultAssigneePool is the hardcoded code owner pool for the scheduled
// sweep. Tempo's CODEOWNERS names only the @grafana/tempo team, and team
// expansion needs an org-scoped token this workflow does not have, so the
// pool is the global "*" owners from before #8133 switched CODEOWNERS to the
// team. Override at runtime with the ASSIGNEE_POOL environment variable (the
// workflow maps the CODEOWNER_ASSIGNEES repository variable into it) or by
// editing this list in the same commit as the CODEOWNERS change.
const defaultAssigneePool = "mdisibio,mapno,yvrhdn,zalegrala,electron0zero,ie-pham,stoewer,javiermolinar,carles-grafana,ruslan-mikhailov,mattdurham,zhxiaogg"

// ownerLogins parses user logins without treating the slash in a team as decoration.
func ownerLogins(value string) map[string]struct{} {
	return parseLogins(value, false)
}

// excludedOwnerLogins accepts optional leading @ and / decorations.
func excludedOwnerLogins(value string) map[string]struct{} {
	return parseLogins(value, true)
}

func parseLogins(value string, stripLeadingSlash bool) map[string]struct{} {
	result := make(map[string]struct{}, strings.Count(value, ",")+1)
	for _, token := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}) {
		token = strings.TrimPrefix(strings.ToLower(token), "@")
		if stripLeadingSlash {
			token = strings.TrimLeft(token, "/")
		}
		if token != "" {
			result[token] = struct{}{}
		}
	}
	return result
}

func hasLogin(set map[string]struct{}, login string) bool {
	_, ok := set[login]
	return ok
}

// resolvePool derives the eligible assignees: the configured pool minus
// EXCLUDED_OWNERS, intersected with the repository's assignable users so a
// stale pool entry cannot trigger a 422 assignment storm.
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

	pool := ownerLogins(cfg.assigneePool)
	excluded := excludedOwnerLogins(cfg.excludedOwners)
	eligible := make(map[string]struct{}, len(pool))
	for login := range pool {
		if !hasLogin(excluded, login) && hasLogin(assignable, login) {
			eligible[login] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(eligible)), nil
}
