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

// Package scrub contains a Scrubber that scrubs error from sensitive details
package scrub // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/scrub"

import (
	"regexp"
)

// Scrubber scrubs error from sensitive details.
type Scrubber interface {
	// Scrub sensitive data from an error.
	Scrub(error) error
}

// replacer structure to store regex matching and replacement functions.
type replacer struct {
	Regex *regexp.Regexp
	Repl  string
}

var _ error = (*scrubbedError)(nil)

// scrubbedError wraps an error and scrubs its `Error()` output.
type scrubbedError struct {
	err      error
	scrubbed string
}

func (s *scrubbedError) Error() string {
	return s.scrubbed
}

func (s *scrubbedError) Unwrap() error {
	return s.err
}

var _ Scrubber = (*scrubber)(nil)

// scrubber scrubs sensitive information from logs
type scrubber struct {
	replacers []replacer
}

func NewScrubber() Scrubber {
	return &scrubber{
		replacers: []replacer{
			// API key as URL parameter (api_key=<API KEY> or apikey=<API KEY>).
			// Any alphanumeric string gets censored, even if not 32 characters long.
			{
				Regex: regexp.MustCompile(`(api_?key=)\b[a-zA-Z0-9]+([a-zA-Z0-9]{5})\b`),
				Repl:  `$1***************************$2`,
			},
			// Application key as URL parameter (api_key=<API KEY> or apikey=<API KEY>).
			// Any alphanumeric string gets censored, even if not 40 characters long.
			{
				Regex: regexp.MustCompile(`(ap(?:p|plication)_?key=)\b[a-zA-Z0-9]+([a-zA-Z0-9]{5})\b`),
				Repl:  `$1***********************************$2`,
			},
			// API key in any place (32 character long alphanumeric ASCII string).
			{
				Regex: regexp.MustCompile(`\b[a-fA-F0-9]{27}([a-fA-F0-9]{5})\b`),
				Repl:  `***************************$1`,
			},
			// Application key in any place (40 character long alphanumeric ASCII string).
			{
				Regex: regexp.MustCompile(`\b[a-fA-F0-9]{35}([a-fA-F0-9]{5})\b`),
				Repl:  `***********************************$1`,
			},
		},
	}
}

func (s *scrubber) Scrub(err error) error {
	if err == nil {
		return nil
	}
	return &scrubbedError{err, s.scrubStr(err.Error())}
}

// Scrub sensitive details from a string.
func (s *scrubber) scrubStr(data string) string {
	for _, repl := range s.replacers {
		data = repl.Regex.ReplaceAllString(data, repl.Repl)
	}
	return data
}
