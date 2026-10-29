// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type minutesArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

// NewMinutesFactory returns a factory for the Minutes OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#minutes
func NewMinutesFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Minutes", &minutesArguments[K]{}, createMinutesFunction[K])
}

func createMinutesFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*minutesArguments[K])

	if !ok {
		return nil, errors.New("MinutesFactory args must be of type *minutesArguments[K]")
	}

	return minutes(args.Duration), nil
}

func minutes[K any](duration ottl.DurationGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Minutes(), nil
	}
}
