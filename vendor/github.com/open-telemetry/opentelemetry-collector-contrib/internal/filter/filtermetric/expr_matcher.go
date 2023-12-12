// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filtermetric // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermetric"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterexpr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
)

type exprMatcher struct {
	matchers []*filterexpr.Matcher
}

func newExprMatcher(expressions []string) (*exprMatcher, error) {
	m := &exprMatcher{}
	for _, expression := range expressions {
		matcher, err := filterexpr.NewMatcher(expression)
		if err != nil {
			return nil, err
		}
		m.matchers = append(m.matchers, matcher)
	}
	return m, nil
}

func (m *exprMatcher) Eval(_ context.Context, tCtx ottlmetric.TransformContext) (bool, error) {
	for _, matcher := range m.matchers {
		matched, err := matcher.MatchMetric(tCtx.GetMetric())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
