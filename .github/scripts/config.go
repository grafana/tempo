package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMarkerLabel = "codeowner-sweep"
	defaultBotActor    = "github-actions[bot]"
	closedLoadDays     = 7
	closedLoadWindow   = closedLoadDays * 24 * time.Hour
)

type config struct {
	repo               string
	assigneePool       string
	excludedOwners     string
	maxAssignments     int
	openLoadMaxAgeDays int
	markerLabel        string
	botActor           string
}

// defaultConfig is shared by environment parsing and tests so new fields cannot
// silently acquire different test and production zero-value semantics.
func defaultConfig(repo string) config {
	return config{
		repo:               repo,
		assigneePool:       "",
		excludedOwners:     "",
		maxAssignments:     25,
		openLoadMaxAgeDays: 90,
		markerLabel:        defaultMarkerLabel,
		botActor:           defaultBotActor,
	}
}

// configFromEnv requires positive assignment and age limits. Values accepted
// here are safe to convert to time.Duration during load expiry checks.
func configFromEnv() (config, error) {
	repo, err := requiredEnv("REPO")
	if err != nil {
		return config{}, err
	}
	if strings.Count(repo, "/") != 1 || strings.HasPrefix(repo, "/") || strings.HasSuffix(repo, "/") {
		return config{}, fmt.Errorf("REPO must use owner/repository form, got %q", repo)
	}
	cfg := defaultConfig(repo)
	cfg.assigneePool = os.Getenv("ASSIGNEE_POOL")
	cfg.excludedOwners = os.Getenv("EXCLUDED_OWNERS")
	cfg.markerLabel = envDefault("MARKER_LABEL", defaultMarkerLabel)
	cfg.botActor = envDefault("BOT_ACTOR", defaultBotActor)
	cfg.maxAssignments, err = envPositiveInt("MAX_ASSIGNMENTS", cfg.maxAssignments)
	if err != nil {
		return config{}, err
	}
	cfg.openLoadMaxAgeDays, err = envPositiveInt("OPEN_LOAD_MAX_AGE_DAYS", cfg.openLoadMaxAgeDays)
	if err != nil {
		return config{}, err
	}
	maxDurationDays := int((time.Duration(1<<63 - 1)) / (24 * time.Hour))
	if cfg.openLoadMaxAgeDays > maxDurationDays {
		return config{}, fmt.Errorf("OPEN_LOAD_MAX_AGE_DAYS must not exceed %d", maxDurationDays)
	}
	return cfg, nil
}

func requiredEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	return value, nil
}

func envDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func envPositiveInt(name string, fallback int) (int, error) {
	value := envDefault(name, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero, got %d", name, parsed)
	}
	return parsed, nil
}
