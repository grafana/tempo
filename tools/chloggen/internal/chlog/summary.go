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

type summary struct {
	Version         string
	BreakingChanges []*Entry
	Deprecations    []*Entry
	NewComponents   []*Entry
	Enhancements    []*Entry
	BugFixes        []*Entry
}

// GenerateSummary generates a changelog entry summary.
func GenerateSummary(version string, entries []*Entry, cfg *config.Config) (string, error) {
	s := summary{
		Version: version,
	}

	for _, entry := range entries {
		switch entry.ChangeType {
		case Breaking:
			s.BreakingChanges = append(s.BreakingChanges, entry)
		case Deprecation:
			s.Deprecations = append(s.Deprecations, entry)
		case NewComponent:
			s.NewComponents = append(s.NewComponents, entry)
		case Enhancement:
			s.Enhancements = append(s.Enhancements, entry)
		case BugFix:
			s.BugFixes = append(s.BugFixes, entry)
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
