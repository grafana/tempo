// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type NanosecondsArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

func NewNanosecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Nanoseconds", &NanosecondsArguments[K]{}, createNanosecondsFunction[K])
}

func createNanosecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*NanosecondsArguments[K])

	if !ok {
		return nil, errors.New("NanosecondsFactory args must be of type *NanosecondsArguments[K]")
	}

	return Nanoseconds(args.Duration)
}

func Nanoseconds[K any](duration ottl.DurationGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Nanoseconds(), nil
	}, nil
}
