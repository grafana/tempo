// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type WeekdayArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewWeekdayFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Weekday", &WeekdayArguments[K]{}, createWeekdayFunction[K])
}

func createWeekdayFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*WeekdayArguments[K])

	if !ok {
		return nil, errors.New("WeekdayFactory args must be of type *WeekdayArguments[K]")
	}

	return Weekday(args.Time)
}

func Weekday[K any](time ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Weekday()), nil
	}, nil
}
