// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type trimPrefixArguments[K any] struct {
	Target ottl.StringGetter[K]
	Prefix ottl.StringGetter[K]
}

// NewTrimPrefixFactory returns a factory for the TrimPrefix OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#trimprefix
func NewTrimPrefixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("TrimPrefix", &trimPrefixArguments[K]{}, createTrimPrefixFunction[K])
}

func createTrimPrefixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*trimPrefixArguments[K])

	if !ok {
		return nil, errors.New("TrimFactory args must be of type *trimPrefixArguments[K]")
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
