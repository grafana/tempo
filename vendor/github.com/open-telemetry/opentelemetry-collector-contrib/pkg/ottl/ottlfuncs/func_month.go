// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type monthArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewMonthFactory returns a factory for the Month OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#month
func NewMonthFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Month", &monthArguments[K]{}, createMonthFunction[K])
}

func createMonthFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*monthArguments[K])

	if !ok {
		return nil, errors.New("MonthFactory args must be of type *monthArguments[K]")
	}

	return month(args.Time), nil
}

func month[K any](time ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Month()), nil
	}
}
