// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type minuteArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewMinuteFactory returns a factory for the Minute OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#minute
func NewMinuteFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Minute", &minuteArguments[K]{}, createMinuteFunction[K])
}

func createMinuteFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*minuteArguments[K])

	if !ok {
		return nil, errors.New("MinuteFactory args must be of type *minuteArguments[K]")
	}

	return minute(args.Time), nil
}

func minute[K any](time ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Minute()), nil
	}
}
