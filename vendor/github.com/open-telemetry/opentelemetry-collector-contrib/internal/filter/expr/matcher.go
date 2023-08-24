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

package expr // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"

import (
	"context"
)

// BoolExpr is an interface that allows matching a context K against a configuration of a match.
type BoolExpr[K any] interface {
	Eval(ctx context.Context, tCtx K) (bool, error)
}

type notMatcher[K any] struct {
	matcher BoolExpr[K]
}

func (nm notMatcher[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	ret, err := nm.matcher.Eval(ctx, tCtx)
	return !ret, err
}

func Not[K any](matcher BoolExpr[K]) BoolExpr[K] {
	return notMatcher[K]{matcher: matcher}
}

type orMatcher[K any] struct {
	matchers []BoolExpr[K]
}

func (om orMatcher[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	for i := range om.matchers {
		ret, err := om.matchers[i].Eval(ctx, tCtx)
		if err != nil {
			return false, err
		}
		if ret {
			return true, nil
		}
	}
	return false, nil
}

func Or[K any](matchers ...BoolExpr[K]) BoolExpr[K] {
	switch len(matchers) {
	case 0:
		return nil
	case 1:
		return matchers[0]
	default:
		return orMatcher[K]{matchers: matchers}
	}
}
