// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type stringArguments[K any] struct {
	Target ottl.StringLikeGetter[K]
}

// NewStringFactory returns a factory for the String OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#string
func NewStringFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("String", &stringArguments[K]{}, createStringFunction[K])
}

func createStringFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*stringArguments[K])

	if !ok {
		return nil, errors.New("StringFactory args must be of type *stringArguments[K]")
	}

	return stringFunc(args.Target), nil
}

func stringFunc[K any](target ottl.StringLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, ok, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
		return value, nil
	}
}
