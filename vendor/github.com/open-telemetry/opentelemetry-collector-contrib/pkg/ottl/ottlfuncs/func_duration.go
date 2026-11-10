// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type durationArguments[K any] struct {
	Duration ottl.StringGetter[K]
}

// NewDurationFactory returns a factory for the Duration OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#duration
func NewDurationFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Duration", &durationArguments[K]{}, createDurationFunction[K])
}

func createDurationFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*durationArguments[K])

	if !ok {
		return nil, errors.New("DurationFactory args must be of type *durationArguments[K]")
	}

	return parseDuration(args.Duration), nil
}

func parseDuration[K any](duration ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		d, err := duration.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		dur, err := time.ParseDuration(d)
		if err != nil {
			return nil, err
		}
		return dur, nil
	}
}
