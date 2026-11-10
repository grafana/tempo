// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type dayArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewDayFactory returns a factory for the Day OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#day
func NewDayFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Day", &dayArguments[K]{}, createDayFunction[K])
}

func createDayFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*dayArguments[K])

	if !ok {
		return nil, errors.New("DayFactory args must be of type *dayArguments[K]")
	}

	return day(args.Time), nil
}

func day[K any](time ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Day()), nil
	}
}
