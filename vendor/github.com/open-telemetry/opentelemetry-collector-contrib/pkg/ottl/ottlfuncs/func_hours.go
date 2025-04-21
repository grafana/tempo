// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type HoursArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

func NewHoursFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Hours", &HoursArguments[K]{}, createHoursFunction[K])
}

func createHoursFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*HoursArguments[K])

	if !ok {
		return nil, errors.New("HoursFactory args must be of type *HoursArguments[K]")
	}

	return Hours(args.Duration)
}

func Hours[K any](duration ottl.DurationGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Hours(), nil
	}, nil
}
