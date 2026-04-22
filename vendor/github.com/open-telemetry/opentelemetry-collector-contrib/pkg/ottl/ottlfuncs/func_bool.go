// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type BoolArguments[K any] struct {
	Target ottl.BoolLikeGetter[K]
}

func NewBoolFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Bool", &BoolArguments[K]{}, createBoolFunction[K])
}

func createBoolFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*BoolArguments[K])

	if !ok {
		return nil, errors.New("BoolFactory args must be of type *BoolArguments[K]")
	}

	return boolFunc(args.Target), nil
}

func boolFunc[K any](target ottl.BoolLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		return *value, nil
	}
}
