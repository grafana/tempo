package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
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
	result := make(map[string]struct{})
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

func readCodeowners(ctx context.Context, client *githubClient, repo string) (string, error) {
	var repository struct {
		DefaultBranch string `json:"default_branch"`
	}
	if _, err := client.request(ctx, http.MethodGet, repositoryEndpoint(repo, ""), nil, nil, &repository); err != nil {
		return "", fmt.Errorf("read repository default branch: %w", err)
	}
	// Fixed GitHub search order; never mutated by callers or tests.
	codeownersPaths := [...]string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"}
	for _, path := range codeownersPaths {
		var content struct {
			Content string `json:"content"`
		}
		query := url.Values{"ref": {repository.DefaultBranch}}
		_, err := client.request(ctx, http.MethodGet, repositoryEndpoint(repo, "/contents/"+path), query, nil, &content)
		if err != nil {
			var statusError *apiError
			if errors.As(err, &statusError) && statusError.status == http.StatusNotFound {
				continue
			}
			return "", fmt.Errorf("read CODEOWNERS candidate %s: %w", path, err)
		}
		decoded, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			return "", fmt.Errorf("decode %s: %w", path, err)
		}
		fmt.Printf("Reading %s from %s.\n", path, repository.DefaultBranch)
		return string(decoded), nil
	}
	return "", errors.New("no CODEOWNERS file found at any known location")
}

func resolvePool(ctx context.Context, client *githubClient, cfg config, groups ownerGroups) ([]string, error) {
	assignees, err := paginateList[user](ctx, client, repositoryEndpoint(cfg.repo, "/assignees"), pageQuery())
	if err != nil {
		return nil, fmt.Errorf("list repository assignees: %w", err)
	}
	assignable := make(map[string]struct{}, len(assignees))
	for _, assignee := range assignees {
		assignable[strings.ToLower(assignee.Login)] = struct{}{}
	}

	pool := make(map[string]struct{}, len(groups.users)+len(cfg.assigneePool))
	if cfg.assigneePool != "" {
		pool = ownerLogins(cfg.assigneePool)
		fmt.Println("Using ASSIGNEE_POOL for the code owner pool.")
	} else {
		for owner := range groups.users {
			pool[owner] = struct{}{}
		}
		org, _, _ := strings.Cut(cfg.repo, "/") // configFromEnv validates owner/repository.
		for _, team := range slices.Sorted(maps.Keys(groups.teams)) {
			_, slug, found := strings.Cut(team, "/")
			if !found || org == "" || slug == "" {
				return nil, fmt.Errorf("invalid team owner %q; expected org/team", team)
			}
			members, err := paginateList[user](ctx, client, "orgs/"+org+"/teams/"+slug+"/members", pageQuery())
			if err != nil {
				fmt.Printf("::warning::Could not expand team @%s: %v. Set CODEOWNER_ASSIGNEES or provide org team-read permission.\n", team, err)
				continue
			}
			for _, member := range members {
				pool[strings.ToLower(member.Login)] = struct{}{}
			}
			fmt.Printf("Expanded @%s to %d member(s).\n", team, len(members))
		}
	}

	excluded := excludedOwnerLogins(cfg.excludedOwners)
	eligible := make(map[string]struct{}, len(pool))
	for owner := range pool {
		if !hasLogin(excluded, owner) && hasLogin(assignable, owner) {
			eligible[owner] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(eligible)), nil
}
