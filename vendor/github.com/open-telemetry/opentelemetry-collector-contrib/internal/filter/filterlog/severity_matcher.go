// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
