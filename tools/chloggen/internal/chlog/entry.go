// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package chlog provides internal functionality for the generation of
// changelogs for OpenTelemetry Go projects.
package chlog

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

const (
	// Breaking is a breaking change.
	Breaking = "breaking"
	// Deprecation is a deprecation change.
	Deprecation = "deprecation"
	// NewComponent is a new component change.
	NewComponent = "new_component"
	// Enhancement is an enhancement change.
	Enhancement = "enhancement"
	// BugFix is a bug fix change.
	BugFix = "bug_fix"
)

// Entry represents a changelog entry.
type Entry struct {
	ChangeLogs []string `yaml:"change_logs"`
	ChangeType string   `yaml:"change_type"`
	Component  string   `yaml:"component"`
	Note       string   `yaml:"note"`
	Issues     []int    `yaml:"issues"`
	SubText    string   `yaml:"subtext"`
	User       string   `yaml:"user"`

	// path is the source file (not part of the on-disk format); BackfillIssues
	// uses it to recover the PR number from git.
	path string
}

// DefaultChangeTypes lists the built-in change types in render order, along with
// the headings used to group them in the generated changelog. It is the value
// used when a configuration does not specify its own 'change_types'.
var DefaultChangeTypes = []config.ChangeType{
	{Key: Breaking, Heading: "🛑 Breaking changes 🛑"},
	{Key: Deprecation, Heading: "🚩 Deprecations 🚩"},
	{Key: NewComponent, Heading: "🚀 New components 🚀"},
	{Key: Enhancement, Heading: "💡 Enhancements 💡"},
	{Key: BugFix, Heading: "🧰 Bug fixes 🧰"},
}

// changeTypeKeys returns the change type keys from the provided change types,
// preserving order.
func changeTypeKeys(changeTypes []config.ChangeType) []string {
	keys := make([]string, 0, len(changeTypes))
	for _, ct := range changeTypes {
		keys = append(keys, ct.Key)
	}
	return keys
}

// Validate validates the changelog entry. The set of valid change types is
// given by changeTypes; when empty, the built-in DefaultChangeTypes are used.
func (e Entry) Validate(requireChangelog bool, components []string, changeTypes []string, validChangeLogs ...string) error {
	if len(changeTypes) == 0 {
		changeTypes = changeTypeKeys(DefaultChangeTypes)
	}
	var errs error
	if requireChangelog && len(e.ChangeLogs) == 0 {
		errs = errors.Join(errs, fmt.Errorf("specify one or more 'change_logs'"))
	}
	for _, cl := range e.ChangeLogs {
		var valid bool
		for _, vcl := range validChangeLogs {
			if cl == vcl {
				valid = true
			}
		}
		if !valid {
			errs = errors.Join(errs, fmt.Errorf("'%s' is not a valid value in 'change_logs'. Specify one of %v", cl, validChangeLogs))
		}
	}

	if !slices.Contains(changeTypes, e.ChangeType) {
		errs = errors.Join(errs, fmt.Errorf("'%s' is not a valid 'change_type'. Specify one of %v", e.ChangeType, changeTypes))
	}

	if strings.TrimSpace(e.Component) == "" {
		errs = errors.Join(errs, fmt.Errorf("specify a 'component'"))
	}

	found := slices.Contains(components, e.Component)
	// only apply component validation if one or more values are present.
	if len(components) > 0 && !found {
		errs = errors.Join(errs, fmt.Errorf("%s is not a valid 'component'. It must be one of %v", e.Component, components))
	}

	if strings.TrimSpace(e.Note) == "" {
		errs = errors.Join(errs, fmt.Errorf("specify a 'note'"))
	}

	// 'issues' is optional; 'update' backfills it from git (see BackfillIssues).

	if strings.TrimSpace(e.User) == "" {
		errs = errors.Join(errs, fmt.Errorf("specify a 'user'"))
	}

	return errs
}

// prSuffix captures the PR number from a squash-merge subject like "title (#1234)".
var prSuffix = regexp.MustCompile(`\(#(\d+)\)\s*$`)

// BackfillIssues fills Issues from the PR number in the commit that added the
// entry file, when the author left 'issues' empty. Best-effort: on any failure
// Issues stays empty and 'update' then fails via MissingIssues.
func (e *Entry) BackfillIssues() {
	if len(e.Issues) > 0 || e.path == "" {
		return
	}
	if pr, ok := prFromGitHistory(e.path); ok {
		e.Issues = []int{pr}
	}
}

// MissingIssues returns the de-duplicated source paths of entries that still
// have no 'issues' after backfill.
func MissingIssues(entriesByChangelog map[string][]*Entry) []string {
	seen := make(map[string]bool)
	var paths []string
	for _, entries := range entriesByChangelog {
		for _, e := range entries {
			if len(e.Issues) == 0 && !seen[e.path] {
				seen[e.path] = true
				paths = append(paths, e.path)
			}
		}
	}
	slices.Sort(paths)
	return paths
}

func prFromGitHistory(path string) (int, bool) {
	// -C <dir> so resolution doesn't depend on the process working directory.
	dir, base := filepath.Dir(path), filepath.Base(path)
	// Only the commit that added the file; a later edit could carry a different PR.
	out, err := exec.Command("git", "-C", dir, "log", "-1", "--diff-filter=A", "--format=%s", "--", base).Output()
	if err != nil {
		return 0, false
	}
	if m := prSuffix.FindSubmatch(bytes.TrimSpace(out)); m != nil {
		if n, err := strconv.Atoi(string(m[1])); err == nil {
			return n, true
		}
	}
	return 0, false
}

// ValidateEntries validates every entry against the configuration, accumulating
// all errors. Both the 'validate' and 'update' commands use this so they
// enforce identical rules (and 'update' never drops or deletes invalid entries).
func ValidateEntries(cfg *config.Config, entriesByChangelog map[string][]*Entry) error {
	changelogRequired := len(cfg.DefaultChangeLogs) == 0
	validChangeLogs := make([]string, 0, len(cfg.ChangeLogs))
	for changeLogKey := range cfg.ChangeLogs {
		validChangeLogs = append(validChangeLogs, changeLogKey)
	}
	validChangeTypes := changeTypeKeys(cfg.ChangeTypes)

	var errs error
	for _, entries := range entriesByChangelog {
		for _, entry := range entries {
			if err := entry.Validate(changelogRequired, cfg.Components, validChangeTypes, validChangeLogs...); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

// ReadEntries reads changelog entries from YAML files based on the provided configuration.
func ReadEntries(cfg *config.Config) (map[string][]*Entry, error) {
	yamlFiles, err := findYamlFiles(cfg.EntriesDir)
	if err != nil {
		return nil, err
	}

	entries := make(map[string][]*Entry)
	for key := range cfg.ChangeLogs {
		entries[key] = make([]*Entry, 0)
	}

	for _, file := range yamlFiles {
		if file == cfg.TemplateYAML || file == cfg.ConfigYAML {
			continue
		}

		fileBytes, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return nil, err
		}

		entry := &Entry{}
		if err = yaml.Unmarshal(fileBytes, entry); err != nil {
			return nil, err
		}
		entry.path = file
		entry.SubText = strings.ReplaceAll(entry.SubText, "\r\n", "\n")
		// 'user' is documented as a handle without the leading '@'; normalize it
		// away so a value like "@octocat" doesn't render as "(@@octocat)".
		entry.User = strings.TrimPrefix(strings.TrimSpace(entry.User), "@")

		if len(entry.ChangeLogs) == 0 {
			for _, cl := range cfg.DefaultChangeLogs {
				entries[cl] = append(entries[cl], entry)
			}
		} else {
			for _, cl := range entry.ChangeLogs {
				if _, ok := entries[cl]; !ok {
					return nil, fmt.Errorf("%s: '%s' is not a valid value in 'change_logs'", file, cl)
				}
				entries[cl] = append(entries[cl], entry)
			}
		}
	}
	return entries, nil
}

// DeleteEntries deletes changelog entries from YAML files based on the provided configuration.
func DeleteEntries(cfg *config.Config) error {
	yamlFiles, err := findYamlFiles(cfg.EntriesDir)
	if err != nil {
		return err
	}

	var errs error
	for _, file := range yamlFiles {
		if file == cfg.TemplateYAML || file == cfg.ConfigYAML {
			continue
		}

		if err := os.Remove(file); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to delete %s: %w", file, err))
		}
	}
	return errs
}

// findYamlFiles finds all YAML files in the specified directory.
// It includes files with both .yaml and .yml extensions.
func findYamlFiles(dir string) ([]string, error) {
	yamlFiles, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find YAML files in %s: %w", dir, err)
	}

	ymlFiles, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find YML files in %s: %w", dir, err)
	}

	return slices.Concat(yamlFiles, ymlFiles), nil
}
