// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TrimPrefixArguments[K any] struct {
	Target ottl.StringGetter[K]
	Prefix ottl.StringGetter[K]
}

func NewTrimPrefixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Trim", &TrimPrefixArguments[K]{}, createTrimPrefixFunction[K])
}

func createTrimPrefixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TrimPrefixArguments[K])

	if !ok {
		return nil, errors.New("TrimFactory args must be of type *TrimPrefixArguments[K]")
	}

	return trimPrefix(args.Target, args.Prefix), nil
}

func trimPrefix[K any](target, prefix ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		prefixVal, err := prefix.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.TrimPrefix(val, prefixVal), nil
	}
}
