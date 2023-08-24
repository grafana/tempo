// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterlog"

import (
	"go.opentelemetry.io/collector/pdata/plog"
)

// severtiyNumberMatcher is a Matcher that matches if the input log record has a severity higher than
// the minSeverityNumber.
type severityNumberMatcher struct {
	matchUndefined    bool
	minSeverityNumber plog.SeverityNumber
}

func newSeverityNumberMatcher(minSeverity plog.SeverityNumber, matchUndefined bool) *severityNumberMatcher {
	return &severityNumberMatcher{
		minSeverityNumber: minSeverity,
		matchUndefined:    matchUndefined,
	}
}

func (snm severityNumberMatcher) match(lr plog.LogRecord) bool {
	// behavior on SeverityNumberUNDEFINED is explicitly defined by matchUndefined
	if lr.SeverityNumber() == plog.SeverityNumberUnspecified {
		return snm.matchUndefined
	}

	// If the log records severity is greater than or equal to the desired severity, it matches
	if lr.SeverityNumber() >= snm.minSeverityNumber {
		return true
	}

	return false
}
