// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IntArguments[K any] struct {
	Target ottl.IntLikeGetter[K]
}

func NewIntFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Int", &IntArguments[K]{}, createIntFunction[K])
}

func createIntFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IntArguments[K])

	if !ok {
		return nil, fmt.Errorf("IntFactory args must be of type *IntArguments[K]")
	}

	return intFunc(args.Target), nil
}

func intFunc[K any](target ottl.IntLikeGetter[K]) ottl.ExprFunc[K] {
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
