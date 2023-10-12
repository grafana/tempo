// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
