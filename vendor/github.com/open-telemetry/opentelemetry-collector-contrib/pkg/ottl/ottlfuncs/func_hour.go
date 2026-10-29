// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type hourArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewHourFactory returns a factory for the Hour OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#hour
func NewHourFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Hour", &hourArguments[K]{}, createHourFunction[K])
}

func createHourFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*hourArguments[K])

	if !ok {
		return nil, errors.New("HourFactory args must be of type *hourArguments[K]")
	}

	return hour(args.Time), nil
}

func hour[K any](t ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		time, err := t.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(time.Hour()), nil
	}
}
