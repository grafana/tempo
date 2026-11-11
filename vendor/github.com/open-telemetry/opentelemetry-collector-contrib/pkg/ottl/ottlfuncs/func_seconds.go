// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type secondsArguments[K any] struct {
	Duration ottl.DurationGetter[K]
}

// NewSecondsFactory returns a factory for the Seconds OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#seconds
func NewSecondsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Seconds", &secondsArguments[K]{}, createSecondsFunction[K])
}

func createSecondsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*secondsArguments[K])

	if !ok {
		return nil, errors.New("SecondsFactory args must be of type *secondsArguments[K]")
	}

	return seconds(args.Duration), nil
}

func seconds[K any](duration ottl.DurationGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return d.Seconds(), nil
	}
}
