package main

import "time"

// Settings for the daily code owner sweep. The script only runs as "go run ."
// from its own gated workflow, so everything here is hardcoded; the only
// per-run values (GH_TOKEN, RUN_URL) stay in the environment.
const (
	repoOwner                 = "grafana"
	repoName                  = "tempo"
	defaultRepo               = repoOwner + "/" + repoName
	defaultMarkerLabel        = "codeowner-sweep"
	defaultBotActor           = "github-actions[bot]"
	defaultMaxAssignments     = 25
	defaultOpenLoadMaxAgeDays = 90
	closedLoadDays            = 7
	closedLoadWindow          = closedLoadDays * 24 * time.Hour

	apiPageSize                    = 100
	apiRequestTimeout              = 30 * time.Second
	maxRequestAttempts             = 3
	maxSecondaryRateLimitRetryTime = 5 * time.Minute
	maxSearchResults               = 1000

	// failureIssueTitle identifies failure issues in both the dedupe search and
	// the created issue; the two must stay in lockstep.
	failureIssueTitle = "assign-codeowner sweep failed"
)

// defaultAssigneePool is the hardcoded code owner pool for the scheduled
// sweep. Tempo's CODEOWNERS names only the @grafana/tempo team, and team
// expansion needs an org-scoped token this workflow does not have, so the pool
// is the global "*" owners from before #8133 switched CODEOWNERS to the team.
// Entries must be lowercase GitHub logins without a leading @; keep them in
// the order you want tie-breaking to consider them. A member who leaves the
// org surfaces as a failed assignment rather than being silently skipped.
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
