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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tools/chloggen/internal/chlog"
	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

const updateUsage = `Usage:
  chloggen update [flags]

Flags:
  -c, --component string   only select entries with this exact component
  -d, --dry                will generate the update text and print to stdout
  -h, --help               help for update
  -v, --version string     will be rendered directly into the update text (default "vTODO")

Global Flags:
      --config string   (optional) chloggen config file`

func TestUpdateErr(t *testing.T) {
	globalCfg = config.New(t.TempDir())
	setupTestDir(t, []*chlog.Entry{})

	var out string
	var err error

	out, err = runCobra(t, "update", "--help")
	assert.Contains(t, out, updateUsage)
	assert.NoError(t, err)

	badEntry, ioErr := os.CreateTemp(globalCfg.EntriesDir, "*.yaml")
	require.NoError(t, ioErr)
	defer badEntry.Close()

	_, ioErr = badEntry.Write([]byte("bad yaml"))
	require.NoError(t, ioErr)
	out, err = runCobra(t, "update")
	assert.Contains(t, out, updateUsage)
	assert.ErrorContains(t, err, "yaml: unmarshal errors")
}

func TestUpdate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows line breaks cause comparison failures w/ golden files.")
	}

	tests := []struct {
		name              string
		entries           []*chlog.Entry
		changeLogs        map[string]string
		defaultChangeLogs []string
		version           string
		dry               bool
		componentFilter   string
	}{
		{
			name:    "all_change_types",
			entries: getSampleEntries(),
			version: "v0.45.0",
		},
		{
			name:    "all_change_types_multiple",
			entries: append(getSampleEntries(), getSampleEntries()...),
			version: "v0.45.0",
		},
		{
			name:    "dry_run",
			entries: getSampleEntries(),
			version: "v0.45.0",
			dry:     true,
		},
		{
			name:    "deprecation_only",
			entries: []*chlog.Entry{deprecationEntry()},
			version: "v0.45.0",
		},
		{
			name:    "new_component_only",
			entries: []*chlog.Entry{newComponentEntry()},
			version: "v0.45.0",
		},
		{
			name:    "bug_fix_only",
			entries: []*chlog.Entry{bugFixEntry()},
			version: "v0.45.0",
		},
		{
			name:    "enhancement_only",
			entries: []*chlog.Entry{enhancementEntry()},
			version: "v0.45.0",
		},
		{
			name:    "breaking_only",
			entries: []*chlog.Entry{breakingEntry()},
			version: "v0.45.0",
		},
		{
			name:    "subtext",
			entries: []*chlog.Entry{entryWithSubtext()},
			version: "v0.45.0",
		},
		{
			name: "multiple_changelogs",
			entries: []*chlog.Entry{
				entryForChangelogs(chlog.Deprecation, 123, "user"),
				entryForChangelogs(chlog.Breaking, 125, "api"),
				entryForChangelogs(chlog.Enhancement, 333, "api", "user"),
				entryForChangelogs(chlog.BugFix, 222, "user"),
				entryForChangelogs(chlog.Deprecation, 223, "api"),
				entryForChangelogs(chlog.BugFix, 111, "api"),
				entryForChangelogs(chlog.Breaking, 11, "user", "api"),
				entryForChangelogs(chlog.Enhancement, 555, "api"),
				entryForChangelogs(chlog.BugFix, 777, "api"),
				entryForChangelogs(chlog.Deprecation, 234, "user", "api"),
				entryForChangelogs(chlog.Enhancement, 21, "user"),
				entryForChangelogs(chlog.BugFix, 32, "user"),
			},
			changeLogs: map[string]string{
				"user": "CHANGELOG.md",
				"api":  "CHANGELOG-API.md",
			},
			version: "v0.45.0",
		},
		{
			name: "multiple_changelogs_single_default",
			entries: []*chlog.Entry{
				entryForChangelogs(chlog.Deprecation, 123),
				entryForChangelogs(chlog.Breaking, 125, "api"),
				entryForChangelogs(chlog.Enhancement, 333, "api", "user"),
				entryForChangelogs(chlog.BugFix, 222),
				entryForChangelogs(chlog.Deprecation, 223, "api"),
				entryForChangelogs(chlog.BugFix, 111, "api"),
				entryForChangelogs(chlog.Breaking, 11, "user", "api"),
				entryForChangelogs(chlog.Enhancement, 555, "api"),
				entryForChangelogs(chlog.BugFix, 777, "api"),
				entryForChangelogs(chlog.Deprecation, 234, "user", "api"),
				entryForChangelogs(chlog.Enhancement, 21),
				entryForChangelogs(chlog.BugFix, 32),
			},
			changeLogs: map[string]string{
				"user": "CHANGELOG.md",
				"api":  "CHANGELOG-API.md",
			},
			defaultChangeLogs: []string{"user"},
			version:           "v0.45.0",
		},
		{
			name: "multiple_changelogs_multiple_defaults",
			entries: []*chlog.Entry{
				entryForChangelogs(chlog.Deprecation, 123),
				entryForChangelogs(chlog.Breaking, 125, "api"),
				entryForChangelogs(chlog.Enhancement, 333, "api", "user"),
				entryForChangelogs(chlog.BugFix, 222),
				entryForChangelogs(chlog.Deprecation, 223, "api"),
				entryForChangelogs(chlog.BugFix, 111, "api"),
				entryForChangelogs(chlog.Breaking, 11, "user", "api"),
				entryForChangelogs(chlog.Enhancement, 555, "api"),
				entryForChangelogs(chlog.BugFix, 777, "api"),
				entryForChangelogs(chlog.Deprecation, 234, "user", "api"),
				entryForChangelogs(chlog.Enhancement, 21),
				entryForChangelogs(chlog.BugFix, 32),
			},
			changeLogs: map[string]string{
				"user": "CHANGELOG.md",
				"api":  "CHANGELOG-API.md",
			},
			defaultChangeLogs: []string{"user", "api"},
			version:           "v0.45.0",
		},
		{
			name:    "filter_component",
			version: "v0.45.0",
			entries: []*chlog.Entry{
				{
					ChangeType: "enhancement",
					Component:  "receiver/foo",
					Note:       "Some change",
					Issues:     []int{1},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/bar",
					Note:       "Some other change",
					Issues:     []int{2},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/foobar",
					Note:       "Some other change for foobar",
					Issues:     []int{3},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/foo",
					Note:       "One more foo change",
					Issues:     []int{4},
					User:       "octocat",
				},
			},
			componentFilter: "receiver/foo",
		},
		{
			name:    "filter_component_no_match",
			version: "v0.45.0",
			entries: []*chlog.Entry{
				{
					ChangeType: "enhancement",
					Component:  "receiver/foo",
					Note:       "Some change",
					Issues:     []int{1},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/bar",
					Note:       "Some other change",
					Issues:     []int{2},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/foobar",
					Note:       "Some other change for foobar",
					Issues:     []int{3},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/foo",
					Note:       "One more foo change",
					Issues:     []int{4},
					User:       "octocat",
				},
			},
			componentFilter: "receiver/foob",
		},
		{
			name: "all_change_types_alphabetical",
			entries: []*chlog.Entry{
				{
					ChangeType: "enhancement",
					Component:  "receiver/a",
					Note:       "Some change",
					Issues:     []int{1},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/bb",
					Note:       "One more bb change",
					Issues:     []int{4},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/b",
					Note:       "Some other change",
					Issues:     []int{3},
					User:       "octocat",
				},
				{
					ChangeType: "enhancement",
					Component:  "receiver/aa",
					Note:       "Some other change for aa",
					Issues:     []int{2},
					User:       "octocat",
				},
			},
			version: "v0.45.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			globalCfg = config.New(tempDir)
			if len(tc.changeLogs) > 0 {
				globalCfg.ChangeLogs = make(map[string]string)
				for key, filename := range tc.changeLogs {
					globalCfg.ChangeLogs[key] = filepath.Join(tempDir, filename)
				}
			}
			if len(tc.defaultChangeLogs) > 0 {
				globalCfg.DefaultChangeLogs = tc.defaultChangeLogs
			}

			setupTestDir(t, tc.entries)

			args := []string{"update", "--version", tc.version}
			if tc.dry {
				args = append(args, "--dry")
			}
			if tc.componentFilter != "" {
				args = append(args, "--component", tc.componentFilter)
			}

			var out string
			var err error

			out, err = runCobra(t, args...)

			assert.Empty(t, err)
			if tc.dry {
				assert.Contains(t, out, "Generated changelog updates for")
			} else {
				for _, filename := range globalCfg.ChangeLogs {
					assert.Contains(t, out, fmt.Sprintf("Finished updating %s", filename))
				}
			}

			for _, filename := range globalCfg.ChangeLogs {
				actualBytes, ioErr := os.ReadFile(filename) // nolint:gosec
				require.NoError(t, ioErr)

				expectedChangelogMD := filepath.Join("testdata", tc.name, filepath.Base(filename))
				expectedBytes, ioErr := os.ReadFile(filepath.Clean(expectedChangelogMD))
				require.NoError(t, ioErr)

				require.Equal(t, string(expectedBytes), string(actualBytes))

				remainingYAMLs, ioErr := filepath.Glob(filepath.Join(globalCfg.EntriesDir, "*.yaml"))
				require.NoError(t, ioErr)
				if tc.dry {
					require.Equal(t, 1+len(tc.entries), len(remainingYAMLs))
				} else {
					require.Equal(t, 1, len(remainingYAMLs))
					require.Equal(t, globalCfg.TemplateYAML, remainingYAMLs[0])
				}
			}
		})
	}
}

func TestUpdateValidatesBeforeWriting(t *testing.T) {
	globalCfg = config.New(t.TempDir())

	// An entry with an invalid change_type must abort update before rendering
	// or deleting, so the source entry file is not lost.
	bad := &chlog.Entry{
		ChangeType: "not_a_real_type",
		Component:  "receiver/foo",
		Note:       "Some change",
		Issues:     []int{1},
		User:       "octocat",
	}
	setupTestDir(t, []*chlog.Entry{bad})

	_, err := runCobra(t, "update", "--version", "v1.0.0")
	require.ErrorContains(t, err, "not_a_real_type")

	// The source entry file (plus TEMPLATE.yaml) must still be present.
	remainingYAMLs, ioErr := filepath.Glob(filepath.Join(globalCfg.EntriesDir, "*.yaml"))
	require.NoError(t, ioErr)
	require.Equal(t, 2, len(remainingYAMLs))
}

func TestUpdateFailsWhenPRUnresolved(t *testing.T) {
	globalCfg = config.New(t.TempDir())

	// A valid entry that left 'issues' empty. The temp dir is not a git repo, so
	// the PR can't be backfilled.
	entry := &chlog.Entry{
		ChangeType: chlog.BugFix,
		Component:  "receiver/foo",
		Note:       "Fix something",
		User:       "octocat",
	}
	setupTestDir(t, []*chlog.Entry{entry})

	// Both --dry and a real update fail rather than emit an entry with no link.
	_, err := runCobra(t, "update", "--dry")
	require.ErrorContains(t, err, "could not determine a PR number")

	_, err = runCobra(t, "update")
	require.ErrorContains(t, err, "could not determine a PR number")

	// The source entry file must not be deleted when the update fails.
	remainingYAMLs, ioErr := filepath.Glob(filepath.Join(globalCfg.EntriesDir, "*.yaml"))
	require.NoError(t, ioErr)
	require.Equal(t, 2, len(remainingYAMLs))
}

func TestUpdateListsAllUnresolvedPRs(t *testing.T) {
	globalCfg = config.New(t.TempDir())

	// Two unresolvable entries plus one with an explicit issue: both unresolved
	// files must appear in a single error, not just the first.
	resolvable := &chlog.Entry{
		ChangeType: chlog.Enhancement,
		Component:  "receiver/foo",
		Note:       "Has a PR",
		Issues:     []int{42},
		User:       "octocat",
	}
	missingA := &chlog.Entry{
		ChangeType: chlog.BugFix,
		Component:  "receiver/foo",
		Note:       "Missing A",
		User:       "octocat",
	}
	missingB := &chlog.Entry{
		ChangeType: chlog.BugFix,
		Component:  "receiver/foo",
		Note:       "Missing B",
		User:       "octocat",
	}
	setupTestDir(t, []*chlog.Entry{resolvable, missingA, missingB})

	_, err := runCobra(t, "update")
	require.ErrorContains(t, err, "could not determine a PR number for 2 changelog")
	// setupTestDir writes entries as 0.yaml (resolvable), 1.yaml, 2.yaml.
	assert.ErrorContains(t, err, "1.yaml")
	assert.ErrorContains(t, err, "2.yaml")
	assert.NotContains(t, err.Error(), "0.yaml")
}
