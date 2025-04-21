// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterlog"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/featuregate"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermatcher"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
)

var useOTTLBridge = featuregate.GlobalRegistry().MustRegister(
	"filter.filterlog.useOTTLBridge",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("When enabled, filterlog will convert filterlog configuration to OTTL and use filterottl evaluation"),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/18642"),
)

// NewSkipExpr creates a BoolExpr that on evaluation returns true if a log should NOT be processed or kept.
// The logic determining if a log should be processed is based on include and exclude settings.
// Include properties are checked before exclude settings are checked.
func NewSkipExpr(mp *filterconfig.MatchConfig) (expr.BoolExpr[ottllog.TransformContext], error) {
	if useOTTLBridge.IsEnabled() {
		return filterottl.NewLogSkipExprBridge(mp)
	}
	var matchers []expr.BoolExpr[ottllog.TransformContext]
	inclExpr, err := newExpr(mp.Include)
	if err != nil {
		return nil, err
	}
	if inclExpr != nil {
		matchers = append(matchers, expr.Not(inclExpr))
	}
	exclExpr, err := newExpr(mp.Exclude)
	if err != nil {
		return nil, err
	}
	if exclExpr != nil {
		matchers = append(matchers, exclExpr)
	}
	return expr.Or(matchers...), nil
}

// propertiesMatcher allows matching a log record against various log record properties.
type propertiesMatcher struct {
	filtermatcher.PropertiesMatcher

	// log bodies to compare to.
	bodyFilters filterset.FilterSet

	// log severity texts to compare to
	severityTextFilters filterset.FilterSet

	// matcher for severity number
	severityNumberMatcher *severityNumberMatcher
}

// NewMatcher creates a LogRecord Matcher that matches based on the given MatchProperties.
func newExpr(mp *filterconfig.MatchProperties) (expr.BoolExpr[ottllog.TransformContext], error) {
	if mp == nil {
		return nil, nil
	}

	if err := mp.ValidateForLogs(); err != nil {
		return nil, err
	}

	rm, err := filtermatcher.NewMatcher(mp)
	if err != nil {
		return nil, err
	}

	var bodyFS filterset.FilterSet
	if len(mp.LogBodies) > 0 {
		bodyFS, err = filterset.CreateFilterSet(mp.LogBodies, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating log record body filters: %w", err)
		}
	}
	var severitytextFS filterset.FilterSet
	if len(mp.LogSeverityTexts) > 0 {
		severitytextFS, err = filterset.CreateFilterSet(mp.LogSeverityTexts, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating log record severity text filters: %w", err)
		}
	}

	pm := &propertiesMatcher{
		PropertiesMatcher:   rm,
		bodyFilters:         bodyFS,
		severityTextFilters: severitytextFS,
	}

	if mp.LogSeverityNumber != nil {
		pm.severityNumberMatcher = newSeverityNumberMatcher(mp.LogSeverityNumber.Min, mp.LogSeverityNumber.MatchUndefined)
	}

	return pm, nil
}

// Eval matches a log record to a set of properties.
// There are 3 sets of properties to match against.
// The log record names are matched, if specified.
// The log record bodies are matched, if specified.
// The attributes are then checked, if specified.
// At least one of log record names or attributes must be specified. It is
// supported to have more than one of these specified, and all specified must
// evaluate to true for a match to occur.
func (mp *propertiesMatcher) Eval(_ context.Context, tCtx ottllog.TransformContext) (bool, error) {
	lr := tCtx.GetLogRecord()
	if mp.bodyFilters != nil && !mp.bodyFilters.Matches(lr.Body().AsString()) {
		return false, nil
	}
	if mp.severityTextFilters != nil && !mp.severityTextFilters.Matches(lr.SeverityText()) {
		return false, nil
	}
	if mp.severityNumberMatcher != nil && !mp.severityNumberMatcher.match(lr) {
		return false, nil
	}

	return mp.Match(lr.Attributes(), tCtx.GetResource(), tCtx.GetInstrumentationScope()), nil
}
