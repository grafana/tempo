// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type secondArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewSecondFactory returns a factory for the Second OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#second
func NewSecondFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Second", &secondArguments[K]{}, createSecondFunction[K])
}

func createSecondFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*secondArguments[K])

	if !ok {
		return nil, errors.New("SecondFactory args must be of type *secondArguments[K]")
	}

	return second(args.Time), nil
}

func second[K any](time ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Second()), nil
	}
}
