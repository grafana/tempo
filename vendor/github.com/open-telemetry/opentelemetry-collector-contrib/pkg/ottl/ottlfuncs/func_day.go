// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type DayArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewDayFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Day", &DayArguments[K]{}, createDayFunction[K])
}

func createDayFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*DayArguments[K])

	if !ok {
		return nil, errors.New("DayFactory args must be of type *DayArguments[K]")
	}

	return Day(args.Time)
}

func Day[K any](time ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Day()), nil
	}, nil
}
