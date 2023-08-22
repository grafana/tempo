// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filtermetric // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermetric"

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
)

// NewSkipExpr creates a BoolExpr that on evaluation returns true if a metric should NOT be processed or kept.
// The logic determining if a metric should be processed is based on include and exclude settings.
// Include properties are checked before exclude settings are checked.
func NewSkipExpr(include *MatchProperties, exclude *MatchProperties) (expr.BoolExpr[ottlmetric.TransformContext], error) {
	var matchers []expr.BoolExpr[ottlmetric.TransformContext]
	inclExpr, err := newExpr(include)
	if err != nil {
		return nil, err
	}
	if inclExpr != nil {
		matchers = append(matchers, expr.Not(inclExpr))
	}
	exclExpr, err := newExpr(exclude)
	if err != nil {
		return nil, err
	}
	if exclExpr != nil {
		matchers = append(matchers, exclExpr)
	}
	return expr.Or(matchers...), nil
}

// NewMatcher constructs a metric Matcher. If an 'expr' match type is specified,
// returns an expr matcher, otherwise a name matcher.
func newExpr(mp *MatchProperties) (expr.BoolExpr[ottlmetric.TransformContext], error) {
	if mp == nil {
		return nil, nil
	}

	if mp.MatchType == Expr {
		if len(mp.Expressions) == 0 {
			return nil, nil
		}
		return newExprMatcher(mp.Expressions)
	}
	if len(mp.MetricNames) == 0 {
		return nil, nil
	}
	return newNameMatcher(mp)
}
