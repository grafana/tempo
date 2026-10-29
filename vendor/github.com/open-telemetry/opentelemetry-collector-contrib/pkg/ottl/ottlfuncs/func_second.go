// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SecondArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewSecondFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Second", &SecondArguments[K]{}, createSecondFunction[K])
}

func createSecondFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SecondArguments[K])

	if !ok {
		return nil, errors.New("SecondFactory args must be of type *SecondArguments[K]")
	}

	return Second(args.Time)
}

func Second[K any](time ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Second()), nil
	}, nil
}
