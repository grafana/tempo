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
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/grafana/tempo/tools/chloggen/internal/config"
)

//go:embed summary.tmpl
var defaultTmpl []byte

// group is one rendered section of the changelog: a heading and the entries
// that belong to a single change type.
type group struct {
	Heading string
	Entries []*Entry
}

type summary struct {
	Version string

	// Groups holds every configured change type in configured order. This is
	// the data model the bundled template ranges over and the only path that
	// surfaces custom change types.
	Groups []group

	// The following fields mirror the built-in change types and are retained so
	// that custom summary templates referencing them by name keep working.
	BreakingChanges []*Entry
	Deprecations    []*Entry
	NewComponents   []*Entry
	Enhancements    []*Entry
	BugFixes        []*Entry
}

// GenerateSummary generates a changelog entry summary. Entries are grouped and
// ordered according to cfg.ChangeTypes; when none are configured, the built-in
// DefaultChangeTypes are used.
func GenerateSummary(version string, entries []*Entry, cfg *config.Config) (string, error) {
	s := summary{
		Version: version,
	}

	changeTypes := cfg.ChangeTypes
	if len(changeTypes) == 0 {
		changeTypes = DefaultChangeTypes
	}

	for _, ct := range changeTypes {
		var grouped []*Entry
		for _, entry := range entries {
			if entry.ChangeType == ct.Key {
				grouped = append(grouped, entry)
			}
		}
		s.Groups = append(s.Groups, group{Heading: ct.Heading, Entries: grouped})

		// Mirror built-in change types onto the named fields for backward compatibility.
		switch ct.Key {
		case Breaking:
			s.BreakingChanges = grouped
		case Deprecation:
			s.Deprecations = grouped
		case NewComponent:
			s.NewComponents = grouped
		case Enhancement:
			s.Enhancements = grouped
		case BugFix:
			s.BugFixes = grouped
		}
	}

	return s.String(cfg.SummaryTemplate)
}

// TemplateFuncMap returns a map of functions to be used in the template.
func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"indent": func(n int, s string) string {
			indent := strings.Repeat(" ", n)
			return indent + strings.ReplaceAll(s, "\n", "\n"+indent)
		},
	}
}

// String renders the summary using the provided template.
func (s summary) String(summaryTemplate string) (string, error) {
	var tmpl *template.Template
	var err error

	if summaryTemplate != "" {
		tmpl, err = template.
			New(filepath.Base(summaryTemplate)).
			Funcs(TemplateFuncMap()).
			Option("missingkey=error").
			ParseFiles(summaryTemplate)
	} else {
		tmpl, err = template.
			New("summary.tmpl").
			Funcs(TemplateFuncMap()).
			Option("missingkey=error").
			Parse(string(defaultTmpl))
	}
	if err != nil {
		return "", err
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, s); err != nil {
		return "", fmt.Errorf("failed executing template: %w", err)
	}

	return buf.String(), nil
}
