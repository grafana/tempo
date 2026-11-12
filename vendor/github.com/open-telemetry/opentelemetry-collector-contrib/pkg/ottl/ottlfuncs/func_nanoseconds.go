// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type nanosecondsArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

// NewNanosecondsFactory returns a factory for the Nanoseconds OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#nanoseconds
func NewNanosecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Nanoseconds", &nanosecondsArguments[K]{}, createNanosecondsFunction[K])
}

func createNanosecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*nanosecondsArguments[K])

	if !ok {
		return nil, errors.New("NanosecondsFactory args must be of type *nanosecondsArguments[K]")
	}

	return nanoseconds(args.Duration), nil
}

func nanoseconds[K any](duration ottl.DurationGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Nanoseconds(), nil
	}
}
