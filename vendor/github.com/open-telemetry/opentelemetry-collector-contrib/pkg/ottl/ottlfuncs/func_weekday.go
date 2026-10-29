// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type weekdayArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewWeekdayFactory returns a factory for the Weekday OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#weekday
func NewWeekdayFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Weekday", &weekdayArguments[K]{}, createWeekdayFunction[K])
}

func createWeekdayFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*weekdayArguments[K])

	if !ok {
		return nil, errors.New("WeekdayFactory args must be of type *weekdayArguments[K]")
	}

	return weekday(args.Time), nil
}

func weekday[K any](time ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Weekday()), nil
	}
}
