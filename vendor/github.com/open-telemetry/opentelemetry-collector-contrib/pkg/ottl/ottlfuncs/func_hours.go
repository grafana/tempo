// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type hoursArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

// NewHoursFactory returns a factory for the Hours OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#hours
func NewHoursFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Hours", &hoursArguments[K]{}, createHoursFunction[K])
}

func createHoursFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*hoursArguments[K])

	if !ok {
		return nil, errors.New("HoursFactory args must be of type *hoursArguments[K]")
	}

	return hours(args.Duration), nil
}

func hours[K any](duration ottl.DurationGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Hours(), nil
	}
}
