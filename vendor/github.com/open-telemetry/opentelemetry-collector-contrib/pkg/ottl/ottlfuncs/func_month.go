// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type MonthArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewMonthFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Month", &MonthArguments[K]{}, createMonthFunction[K])
}

func createMonthFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MonthArguments[K])

	if !ok {
		return nil, fmt.Errorf("MonthFactory args must be of type *MonthArguments[K]")
	}

	return Month(args.Time)
}

func Month[K any](time ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Month()), nil
	}, nil
}
