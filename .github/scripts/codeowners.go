package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/google/go-github/v93/github"
)

type ownerKind uint8

const (
	ownerUser ownerKind = iota
	ownerTeam
)

type ownerGroups struct {
	users    map[string]struct{}
	teams    map[string]struct{}
	partial  map[string]struct{}
	excluded map[string]struct{}
}

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

func parseCodeowners(text string, excluded string) ownerGroups {
	estimatedOwners := strings.Count(text, "\n") + 1
	groups := ownerGroups{
		users:    make(map[string]struct{}, estimatedOwners),
		teams:    make(map[string]struct{}, estimatedOwners),
		partial:  make(map[string]struct{}, estimatedOwners),
		excluded: make(map[string]struct{}, estimatedOwners),
	}
	counts := map[ownerKind]map[string]int{
		ownerUser: make(map[string]int, estimatedOwners),
		ownerTeam: make(map[string]int, estimatedOwners),
	}
	catchAll := make(map[string]struct{}, estimatedOwners)
	rules := 0
	for line := range strings.Lines(text) {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		rules++
		seen := make(map[string]struct{}, len(fields)-1)
		for _, field := range fields[1:] {
			if !strings.HasPrefix(field, "@") {
				continue
			}
			owner := strings.ToLower(strings.TrimPrefix(field, "@"))
			if _, ok := seen[owner]; ok {
				continue
			}
			seen[owner] = struct{}{}
			if fields[0] == "*" {
				catchAll[owner] = struct{}{}
			}
			kind := ownerUser
			if strings.Contains(owner, "/") {
				kind = ownerTeam
			}
			counts[kind][owner]++
		}
	}

	excludedOwners := excludedOwnerLogins(excluded)
	for _, kind := range []ownerKind{ownerUser, ownerTeam} {
		for owner, count := range counts[kind] {
			switch {
			case hasLogin(excludedOwners, owner):
				groups.excluded[owner] = struct{}{}
			case count == rules || hasLogin(catchAll, owner):
				if kind == ownerUser {
					groups.users[owner] = struct{}{}
				} else {
					groups.teams[owner] = struct{}{}
				}
			default:
				// Owners matching only some rules cannot own an arbitrary PR.
				groups.partial[owner] = struct{}{}
			}
		}
	}
	return groups
}

func hasLogin(set map[string]struct{}, login string) bool {
	_, ok := set[login]
	return ok
}

func readCodeowners(ctx context.Context, client *github.Client, repo string) (string, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return "", err
	}
	repository, _, err := client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return "", fmt.Errorf("read repository default branch: %w", err)
	}
	// GitHub searches these locations in this order and uses the first that
	// exists: .github/, the root, then docs/.
	codeownersPaths := [...]string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}
	for _, path := range codeownersPaths {
		content, _, _, err := client.Repositories.GetContents(ctx, owner, name, path, &github.RepositoryContentGetOptions{Ref: repository.GetDefaultBranch()})
		if err != nil {
			var statusError *github.ErrorResponse
			if errors.As(err, &statusError) && statusError.Response.StatusCode == http.StatusNotFound {
				continue
			}
			return "", fmt.Errorf("read CODEOWNERS candidate %s: %w", path, err)
		}
		if content == nil {
			continue // a directory lives at this path
		}
		decoded, err := content.GetContent()
		if err != nil {
			return "", fmt.Errorf("decode %s: %w", path, err)
		}
		fmt.Printf("Reading %s from %s.\n", path, repository.GetDefaultBranch())
		return decoded, nil
	}
	return "", errors.New("no CODEOWNERS file found at any known location")
}

func resolvePool(ctx context.Context, client *github.Client, cfg config, groups ownerGroups) ([]string, error) {
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

	pool := make(map[string]struct{}, len(groups.users)+len(cfg.assigneePool))
	if cfg.assigneePool != "" {
		for login := range ownerLogins(cfg.assigneePool) {
			pool[login] = struct{}{}
		}
		fmt.Println("Using ASSIGNEE_POOL for the code owner pool.")
	} else {
		for groupOwner := range groups.users {
			pool[groupOwner] = struct{}{}
		}
		for _, team := range slices.Sorted(maps.Keys(groups.teams)) {
			// Expand the team under its own org, not the repository's: a
			// CODEOWNERS entry may legitimately name a team from another org.
			teamOrg, slug, found := strings.Cut(team, "/")
			if !found || teamOrg == "" || slug == "" {
				return nil, fmt.Errorf("invalid team owner %q; expected org/team", team)
			}
			members, err := eachPage(ctx, func(opts *github.ListOptions) ([]*github.User, *github.Response, error) {
				return client.Teams.ListTeamMembersBySlug(ctx, teamOrg, slug, &github.TeamListTeamMembersOptions{ListOptions: *opts})
			})
			if err != nil {
				fmt.Printf("::warning::Could not expand team @%s: %v. Set ASSIGNEE_POOL or provide org team-read permission.\n", team, err)
				continue
			}
			for _, member := range members {
				pool[strings.ToLower(member.GetLogin())] = struct{}{}
			}
			fmt.Printf("Expanded @%s to %d member(s).\n", team, len(members))
		}
	}

	excluded := excludedOwnerLogins(cfg.excludedOwners)
	eligible := make(map[string]struct{}, len(pool))
	for groupOwner := range pool {
		if !hasLogin(excluded, groupOwner) && hasLogin(assignable, groupOwner) {
			eligible[groupOwner] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(eligible)), nil
}
