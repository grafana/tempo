package main

import "time"

// Settings for the daily code owner sweep. The script only runs as "go run ."
// from its own gated workflow, so everything here is hardcoded; the only
// per-run values (GH_TOKEN, RUN_URL) stay in the environment.
const (
	defaultRepo               = "grafana/tempo"
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
