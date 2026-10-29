// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type millisecondsArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

// NewMillisecondsFactory returns a factory for the Milliseconds OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#milliseconds
func NewMillisecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Milliseconds", &millisecondsArguments[K]{}, createMillisecondsFunction[K])
}

func createMillisecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*millisecondsArguments[K])

	if !ok {
		return nil, errors.New("MillisecondsFactory args must be of type *millisecondsArguments[K]")
	}

	return milliseconds(args.Duration), nil
}

func milliseconds[K any](duration ottl.DurationGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Milliseconds(), nil
	}
}
