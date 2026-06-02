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

package chlog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

func TestEntry(t *testing.T) {
	tmpl := template.Must(
		template.
			New("summary.tmpl").
			Funcs(TemplateFuncMap()).
			Option("missingkey=error").
			Parse(string(defaultTmpl)))

	testCases := []struct {
		name             string
		entry            Entry
		requireChangeLog bool
		validChangeLogs  []string
		components       []string
		expectErr        string
		toString         string
	}{
		{
			name:      "empty",
			entry:     Entry{},
			expectErr: "'' is not a valid 'change_type'. Specify one of [breaking deprecation new_component enhancement bug_fix]\nspecify a 'component'\nspecify a 'note'\nspecify one or more issues #'s",
		},
		{
			name: "missing_component",
			entry: Entry{
				ChangeType: "enhancement",
				Note:       "enhance!",
				Issues:     []int{123},
				SubText:    "",
			},
			expectErr: "specify a 'component'",
		},
		{
			name: "missing_note",
			entry: Entry{
				ChangeType: "bug_fix",
				Component:  "bar",
				Issues:     []int{123},
				SubText:    "",
			},
			expectErr: "specify a 'note'",
		},
		{
			name: "missing_issue",
			entry: Entry{
				ChangeType: "bug_fix",
				Component:  "bar",
				Note:       "fix bar",
				SubText:    "",
			},
			expectErr: "specify one or more issues #'s",
		},
		{
			name: "missing_required_changelog",
			entry: Entry{
				ChangeType: "bug_fix",
				Component:  "bar",
				Note:       "fix bar",
				Issues:     []int{123},
				SubText:    "",
			},
			requireChangeLog: true,
			validChangeLogs:  []string{"foo"},
			expectErr:        "specify one or more 'change_logs'",
		},
		{
			name: "invalid_changelog",
			entry: Entry{
				ChangeLogs: []string{"bar"},
				ChangeType: "bug_fix",
				Component:  "bar",
				Note:       "fix bar",
				Issues:     []int{123},
				SubText:    "",
			},
			validChangeLogs: []string{"foo"},
			expectErr:       "'bar' is not a valid value in 'change_logs'. Specify one of [foo]",
		},
		{
			name: "valid",
			entry: Entry{
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "",
			},
			toString: "- `foo`: broke foo (#123)",
		},
		{
			name: "multiple_issues",
			entry: Entry{
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123, 345},
				SubText:    "",
			},
			toString: "- `foo`: broke foo (#123, #345)",
		},
		{
			name: "subtext",
			entry: Entry{
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "more details",
			},
			toString: "- `foo`: broke foo (#123)\n  more details",
		},
		{
			name: "multiline subtext",
			entry: Entry{
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "more details\nsecond line",
			},
			toString: "- `foo`: broke foo (#123)\n  more details\n  second line",
		},
		{
			name: "required_changelog",
			entry: Entry{
				ChangeLogs: []string{"foo"},
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "more details",
			},
			requireChangeLog: true,
			validChangeLogs:  []string{"foo"},
			toString:         "- `foo`: broke foo (#123)\n  more details",
		},
		{
			name: "default_changelog",
			entry: Entry{
				ChangeLogs: []string{"foo"},
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "more details",
			},
			requireChangeLog: false,
			validChangeLogs:  []string{"foo"},
			toString:         "- `foo`: broke foo (#123)\n  more details",
		},
		{
			name: "subset_of_changelogs",
			entry: Entry{
				ChangeLogs: []string{"foo", "bar"},
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "more details",
			},
			validChangeLogs: []string{"foo", "bar", "baz"},
			toString:        "- `foo`: broke foo (#123)\n  more details",
		},
		{
			name: "all_changelogs",
			entry: Entry{
				ChangeLogs: []string{"foo", "bar"},
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "more details",
			},
			validChangeLogs: []string{"foo", "bar"},
			toString:        "- `foo`: broke foo (#123)\n  more details",
		},
		{
			name: "all_changelogs",
			entry: Entry{
				ChangeLogs: []string{"foo", "bar"},
				ChangeType: "breaking",
				Component:  "foo",
				Note:       "broke foo",
				Issues:     []int{123},
				SubText:    "more details",
			},
			validChangeLogs: []string{"foo", "bar"},
			toString:        "- `foo`: broke foo (#123)\n  more details",
		},
		{
			name: "with_components",
			entry: Entry{
				ChangeLogs: []string{"foo", "bar"},
				ChangeType: "enhancement",
				Component:  "foo",
				Note:       "changed foo",
				Issues:     []int{123},
				SubText:    "more details",
			},
			components:      []string{"bar"},
			validChangeLogs: []string{"foo", "bar"},
			toString:        "- `foo`: changed foo (#123)\n  more details",
			expectErr:       "foo is not a valid 'component'. It must be one of [bar]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.entry.Validate(tc.requireChangeLog, tc.components, tc.validChangeLogs...)
			if tc.expectErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectErr, err.Error())
				return
			}
			assert.NoError(t, err)

			buf := bytes.Buffer{}
			err = tmpl.ExecuteTemplate(&buf, "entry", tc.entry)
			assert.NoError(t, err)
			assert.Equal(t, tc.toString, buf.String())
		})
	}
}

func TestReadDeleteEntries(t *testing.T) {
	tempDir := t.TempDir()
	entriesDir := filepath.Join(tempDir, config.DefaultEntriesDir)
	require.NoError(t, os.Mkdir(entriesDir, 0o750))

	entryA := Entry{
		ChangeLogs: []string{"foo"},
		ChangeType: "breaking",
		Component:  "foo",
		Note:       "broke foo",
		Issues:     []int{123},
	}
	writeEntry(t, entriesDir, &entryA, "yaml")

	entryB := Entry{
		ChangeLogs: []string{"bar"},
		ChangeType: "bug_fix",
		Component:  "bar",
		Note:       "fix bar",
		Issues:     []int{345, 678},
		SubText:    "more details",
	}
	writeEntry(t, entriesDir, &entryB, "yml")

	entryC := Entry{
		ChangeLogs: []string{},
		ChangeType: "enhancement",
		Component:  "other",
		Note:       "enhance!",
		Issues:     []int{555},
	}
	writeEntry(t, entriesDir, &entryC, "yaml")

	entryD := Entry{
		ChangeLogs: []string{"foo", "bar"},
		ChangeType: "deprecation",
		Component:  "foobar",
		Note:       "deprecate something",
		Issues:     []int{999},
	}
	writeEntry(t, entriesDir, &entryD, "yml")

	// Put config and template files in entries_dir to ensure they are ignored when reading/deleting entries
	configYAML, err := os.Create(filepath.Join(entriesDir, "config.yaml")) //nolint:gosec
	require.NoError(t, err)
	defer configYAML.Close()

	templateYAML, err := os.Create(filepath.Join(entriesDir, "TEMPLATE.yaml")) //nolint:gosec
	require.NoError(t, err)
	defer templateYAML.Close()

	cfg := &config.Config{
		ConfigYAML:   configYAML.Name(),
		TemplateYAML: templateYAML.Name(),
		ChangeLogs: map[string]string{
			"foo": filepath.Join(entriesDir, "CHANGELOG.foo.md"),
			"bar": filepath.Join(entriesDir, "CHANGELOG.bar.md"),
		},
		DefaultChangeLogs: []string{"foo"},
		EntriesDir:        entriesDir,
	}

	changeLogEntries, err := ReadEntries(cfg)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(changeLogEntries))

	assert.Contains(t, changeLogEntries, "foo")
	assert.Contains(t, changeLogEntries, "bar")

	assert.ElementsMatch(t, []*Entry{&entryA, &entryC, &entryD}, changeLogEntries["foo"])
	assert.ElementsMatch(t, []*Entry{&entryB, &entryD}, changeLogEntries["bar"])

	assert.NoError(t, DeleteEntries(cfg))
	changeLogEntries, err = ReadEntries(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(changeLogEntries))
	assert.Empty(t, changeLogEntries["foo"])
	assert.Empty(t, changeLogEntries["bar"])

	// Ensure these weren't deleted
	assert.FileExists(t, cfg.ConfigYAML)
	assert.FileExists(t, cfg.TemplateYAML)
}

func TestFindYamlFilesEmptyDir(t *testing.T) {
	dir := t.TempDir()

	files, err := findYamlFiles(dir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestFindYamlFilesBothExtensions(t *testing.T) {
	dir := t.TempDir()

	yamlFile, err := os.Create(filepath.Join(dir, "one.yaml")) //nolint:gosec
	require.NoError(t, err)
	defer yamlFile.Close()

	ymlFile, err := os.Create(filepath.Join(dir, "two.yml")) //nolint:gosec
	require.NoError(t, err)
	defer ymlFile.Close()

	// Non-YAML file should be ignored
	txtFile, err := os.Create(filepath.Join(dir, "ignore.txt")) //nolint:gosec
	require.NoError(t, err)
	defer txtFile.Close()

	files, err := findYamlFiles(dir)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{yamlFile.Name(), ymlFile.Name()}, files)
}

func TestFindYamlFilesNonRecursive(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML file in the root dir
	rootYAML, err := os.Create(filepath.Join(dir, "root.yaml")) //nolint:gosec
	require.NoError(t, err)
	defer rootYAML.Close()

	// Create a subdirectory with a YAML file inside it
	subdir := filepath.Join(dir, "nested")
	require.NoError(t, os.Mkdir(subdir, 0o750))

	nestedYML, err := os.Create(filepath.Join(subdir, "nested.yml")) //nolint:gosec
	require.NoError(t, err)
	defer nestedYML.Close()

	files, err := findYamlFiles(dir)
	require.NoError(t, err)
	// Should only include files directly under dir, not nested ones
	assert.ElementsMatch(t, []string{rootYAML.Name()}, files)
}

func writeEntry(t *testing.T, dir string, entry *Entry, ext string) {
	require.Contains(t, []string{"yaml", "yml"}, ext, "ext must be 'yaml' or 'yml'")

	entryBytes, err := yaml.Marshal(entry)
	require.NoError(t, err)

	entryFile, err := os.CreateTemp(dir, "*."+ext)
	require.NoError(t, err)
	defer entryFile.Close()

	_, err = entryFile.Write(entryBytes)
	require.NoError(t, err)
}
