// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type MillisecondsArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

func NewMillisecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Milliseconds", &MillisecondsArguments[K]{}, createMillisecondsFunction[K])
}

func createMillisecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MillisecondsArguments[K])

	if !ok {
		return nil, errors.New("MillisecondsFactory args must be of type *MillisecondsArguments[K]")
	}

	return Milliseconds(args.Duration)
}

func Milliseconds[K any](duration ottl.DurationGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Milliseconds(), nil
	}, nil
}
