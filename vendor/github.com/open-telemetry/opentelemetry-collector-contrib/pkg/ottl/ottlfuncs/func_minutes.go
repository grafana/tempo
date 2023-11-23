// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type MinutesArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

func NewMinutesFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Minutes", &MinutesArguments[K]{}, createMinutesFunction[K])
}
func createMinutesFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MinutesArguments[K])

	if !ok {
		return nil, fmt.Errorf("MinutesFactory args must be of type *MinutesArguments[K]")
	}

	return Minutes(args.Duration)
}

func Minutes[K any](duration ottl.DurationGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Minutes(), nil
	}, nil
}
