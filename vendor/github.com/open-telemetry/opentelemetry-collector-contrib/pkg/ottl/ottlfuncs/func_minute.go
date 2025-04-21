// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type MinuteArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewMinuteFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Minute", &MinuteArguments[K]{}, createMinuteFunction[K])
}

func createMinuteFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MinuteArguments[K])

	if !ok {
		return nil, errors.New("MinuteFactory args must be of type *MinuteArguments[K]")
	}

	return Minute(args.Time)
}

func Minute[K any](time ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Minute()), nil
	}, nil
}
