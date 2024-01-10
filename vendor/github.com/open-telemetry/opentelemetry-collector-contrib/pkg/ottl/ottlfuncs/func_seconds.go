// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SecondsArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

func NewSecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Seconds", &SecondsArguments[K]{}, createSecondsFunction[K])
}
func createSecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SecondsArguments[K])

	if !ok {
		return nil, fmt.Errorf("SecondsFactory args must be of type *SecondsArguments[K]")
	}

	return Seconds(args.Duration)
}

func Seconds[K any](duration ottl.DurationGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Seconds(), nil
	}, nil
}
