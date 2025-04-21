// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type StringArguments[K any] struct {
	Target ottl.StringLikeGetter[K]
}

func NewStringFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("String", &StringArguments[K]{}, createStringFunction[K])
}

func createStringFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*StringArguments[K])

	if !ok {
		return nil, errors.New("StringFactory args must be of type *StringArguments[K]")
	}

	return stringFunc(args.Target), nil
}

func stringFunc[K any](target ottl.StringLikeGetter[K]) ottl.ExprFunc[K] {
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
