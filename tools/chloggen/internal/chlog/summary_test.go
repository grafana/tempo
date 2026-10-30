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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

func TestSummary(t *testing.T) {
	brk1 := Entry{
		ChangeType: Breaking,
		Component:  "foo",
		Note:       "broke foo",
		Issues:     []int{123},
	}
	brk2 := Entry{
		ChangeType: Breaking,
		Component:  "bar",
		Note:       "broke bar",
		Issues:     []int{345, 678},
		SubText:    "more details",
	}
	dep1 := Entry{
		ChangeType: Deprecation,
		Component:  "foo",
		Note:       "deprecate foo",
		Issues:     []int{1234},
	}
	dep2 := Entry{
		ChangeType: Deprecation,
		Component:  "bar",
		Note:       "deprecate bar",
		Issues:     []int{3456, 6789},
		SubText:    "more details",
	}
	enh1 := Entry{
		ChangeType: Enhancement,
		Component:  "foo",
		Note:       "enhance foo",
		Issues:     []int{12},
	}
	enh2 := Entry{
		ChangeType: Enhancement,
		Component:  "bar",
		Note:       "enhance bar",
		Issues:     []int{34, 67},
		SubText:    "more details",
	}
	bug1 := Entry{
		ChangeType: BugFix,
		Component:  "foo",
		Note:       "bug foo",
		Issues:     []int{1},
	}
	bug2 := Entry{
		ChangeType: BugFix,
		Component:  "bar",
		Note:       "bug bar",
		Issues:     []int{3, 6},
		SubText:    "more details",
	}
	new1 := Entry{
		ChangeType: NewComponent,
		Component:  "foo",
		Note:       "new foo",
		Issues:     []int{2},
	}
	new2 := Entry{
		ChangeType: NewComponent,
		Component:  "bar",
		Note:       "new bar",
		Issues:     []int{4, 7},
		SubText:    "more details",
	}

	actual, err := GenerateSummary("1.0", []*Entry{&brk1, &brk2, &dep1, &dep2, &enh1, &enh2, &bug1, &bug2, &new1, &new2}, &config.Config{})
	assert.NoError(t, err)

	// This file is not meant to be the entire changelog so will not pass markdownlint if named with .md extension.
	expected, err := os.ReadFile(filepath.Join("testdata", "CHANGELOG"))
	require.NoError(t, err)

	assert.Equal(t, string(expected), actual)
}

func TestConfiguredChangeTypes(t *testing.T) {
	sec := Entry{
		ChangeType: "security",
		Component:  "foo",
		Note:       "fix vuln",
		Issues:     []int{1},
	}
	enh := Entry{
		ChangeType: Enhancement,
		Component:  "bar",
		Note:       "improve bar",
		Issues:     []int{2},
	}

	cfg := &config.Config{
		ChangeTypes: []config.ChangeType{
			{Key: "security", Heading: "Security"},
			{Key: Enhancement, Heading: "Enhancements"},
		},
	}

	// enh is passed before sec, but the configured order must drive the output.
	actual, err := GenerateSummary("1.0", []*Entry{&enh, &sec}, cfg)
	require.NoError(t, err)

	// The custom 'security' change type renders (the default template would drop it).
	assert.Contains(t, actual, "### Security")
	assert.Contains(t, actual, "- `foo`: fix vuln (#1)")
	assert.Contains(t, actual, "### Enhancements")
	assert.Contains(t, actual, "- `bar`: improve bar (#2)")

	// Sections appear in the configured order, not entry order or built-in order.
	assert.Less(t, strings.Index(actual, "### Security"), strings.Index(actual, "### Enhancements"))
}

func TestIndentSubtext(t *testing.T) {
	indent := TemplateFuncMap()["indent"].(func(int, string) string)
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "empty"},
		{name: "yaml trailing newline", text: "Details.\n", want: "  Details."},
		{name: "multiple trailing newlines", text: "Details.\n\n", want: "  Details."},
		{name: "paragraphs", text: "First.\n \t\nSecond.\n", want: "  First.\n\n  Second."},
		{name: "nested list", text: "Details:\n* Item\n  * Nested\n", want: "  Details:\n  * Item\n    * Nested"},
		{name: "hard break", text: "First.  \nSecond.\n", want: "  First.  \n  Second."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, indent(2, tt.text))
		})
	}
}

func TestTempoSummarySpacing(t *testing.T) {
	cfg := &config.Config{
		SummaryTemplate: filepath.Join("..", "..", "..", "..", ".chloggen", "summary.tmpl"),
		ChangeTypes: []config.ChangeType{
			{Key: "feature", Heading: "Features"},
			{Key: BugFix, Heading: "Bug fixes"},
		},
	}
	entries := []*Entry{
		{ChangeType: "feature", Component: "tempo", Note: "Add a feature.", Issues: []int{1}, User: "octocat", SubText: "First paragraph.\n\nSecond paragraph.\n"},
		{ChangeType: "feature", Component: "tempo", Note: "Add another feature.", Issues: []int{2}, User: "octocat"},
		{ChangeType: BugFix, Component: "tempo", Note: "Fix a bug.", Issues: []int{3}, User: "octocat", SubText: "Details.\n"},
	}
	got, err := GenerateSummary("v3.1.0-rc.0", entries, cfg)
	require.NoError(t, err)
	want := "\n# v3.1.0-rc.0\n\n## Features\n\n" +
		"- `tempo`: Add a feature. ([#1](https://github.com/grafana/tempo/issues/1)) (@octocat)\n" +
		"  First paragraph.\n\n  Second paragraph.\n" +
		"- `tempo`: Add another feature. ([#2](https://github.com/grafana/tempo/issues/2)) (@octocat)\n\n" +
		"## Bug fixes\n\n" +
		"- `tempo`: Fix a bug. ([#3](https://github.com/grafana/tempo/issues/3)) (@octocat)\n  Details.\n"
	assert.Equal(t, want, got)
}

func TestCustomSummary(t *testing.T) {
	brk1 := Entry{
		ChangeType: Breaking,
		Component:  "foo",
		Note:       "broke foo",
		Issues:     []int{123},
	}
	brk2 := Entry{
		ChangeType: Breaking,
		Component:  "bar",
		Note:       "broke bar",
		Issues:     []int{345, 678},
		SubText:    "more details",
	}

	actual, err := GenerateSummary("1.0", []*Entry{&brk1, &brk2}, &config.Config{SummaryTemplate: filepath.Join("testdata", "custom.tmpl")})
	require.NoError(t, err)

	// This file is not meant to be the entire changelog so will not pass markdownlint if named with .md extension.
	expected, err := os.ReadFile(filepath.Join("testdata", "CHANGELOG_custom"))
	require.NoError(t, err)

	assert.Equal(t, string(expected), actual)
}
