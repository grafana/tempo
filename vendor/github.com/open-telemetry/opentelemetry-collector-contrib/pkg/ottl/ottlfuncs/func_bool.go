// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type boolArguments[K any] struct {
	Target ottl.BoolLikeGetter[K]
}

// NewBoolFactory returns a factory for the Bool OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#bool
func NewBoolFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Bool", &boolArguments[K]{}, createBoolFunction[K])
}

func createBoolFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*boolArguments[K])

	if !ok {
		return nil, errors.New("BoolFactory args must be of type *boolArguments[K]")
	}

	return boolFunc(args.Target), nil
}

func boolFunc[K any](target ottl.BoolLikeGetter[K]) ottl.ExprFunc[K] {
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
