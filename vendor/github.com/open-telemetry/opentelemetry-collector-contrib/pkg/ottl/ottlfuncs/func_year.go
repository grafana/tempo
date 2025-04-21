// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type YearArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewYearFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Year", &YearArguments[K]{}, createYearFunction[K])
}

func createYearFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*YearArguments[K])

	if !ok {
		return nil, errors.New("YearFactory args must be of type *YearArguments[K]")
	}

	return Year(args.Time)
}

func Year[K any](time ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Year()), nil
	}, nil
}
