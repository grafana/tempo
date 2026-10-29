// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type hasPrefixArguments[K any] struct {
	Target ottl.StringGetter[K]
	Prefix ottl.StringGetter[K]
}

// NewHasPrefixFactory returns a factory for the HasPrefix OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#hasprefix
func NewHasPrefixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("HasPrefix", &hasPrefixArguments[K]{}, createHasPrefixFunction[K])
}

func createHasPrefixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*hasPrefixArguments[K])

	if !ok {
		return nil, errors.New("HasPrefixFactory args must be of type *hasPrefixArguments[K]")
	}

	return hasPrefix(args.Target, args.Prefix), nil
}

func hasPrefix[K any](target, prefix ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		prefixVal, err := prefix.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.HasPrefix(val, prefixVal), nil
	}
}
