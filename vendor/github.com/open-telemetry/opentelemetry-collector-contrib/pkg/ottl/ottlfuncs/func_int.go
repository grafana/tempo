// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type intArguments[K any] struct {
	Target ottl.IntLikeGetter[K]
}

// NewIntFactory returns a factory for the Int OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#int
func NewIntFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Int", &intArguments[K]{}, createIntFunction[K])
}

func createIntFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*intArguments[K])

	if !ok {
		return nil, errors.New("IntFactory args must be of type *intArguments[K]")
	}

	return intFunc(args.Target), nil
}

func intFunc[K any](target ottl.IntLikeGetter[K]) ottl.ExprFunc[K] {
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
