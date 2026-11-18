package main

import (
	"fmt"
	"os"
	"slices"
	"time"
)

// Hardcoded settings for the daily sweep. The script only runs as
// "go run ." from its own workflow on grafana/tempo (the job is gated on
// github.repository), so nothing here needs to be configurable at runtime:
// changing a setting is a code change made in the same commit as the workflow
// or CODEOWNERS change it coordinates with. The only genuinely per-run values
// stay in the environment: GH_TOKEN and, for the failure reporter, RUN_URL.
const (
	defaultRepo               = "grafana/tempo"
	defaultMarkerLabel        = "codeowner-sweep"
	defaultBotActor           = "github-actions[bot]"
	defaultMaxAssignments     = 25
	defaultOpenLoadMaxAgeDays = 90
	closedLoadDays            = 7
	closedLoadWindow          = closedLoadDays * 24 * time.Hour
)

// defaultExcludedOwners never get assigned even if they appear in the pool.
var defaultExcludedOwners = []string{"grafana/docs-traces"}

// config carries sweep settings. A zero-value config is not usable:
// maxAssignments and openLoadMaxAgeDays must be positive and markerLabel and
// botActor must be non-empty. Every production path uses defaultConfig.
type config struct {
	repo               string
	assigneePool       []string
	excludedOwners     []string
	maxAssignments     int
	openLoadMaxAgeDays int
	markerLabel        string
	botActor           string
}

func defaultConfig() config {
	return config{
		repo:               defaultRepo,
		assigneePool:       slices.Clone(defaultAssigneePool),
		excludedOwners:     slices.Clone(defaultExcludedOwners),
		maxAssignments:     defaultMaxAssignments,
		openLoadMaxAgeDays: defaultOpenLoadMaxAgeDays,
		markerLabel:        defaultMarkerLabel,
		botActor:           defaultBotActor,
	}
}

func requiredEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	return value, nil
}
