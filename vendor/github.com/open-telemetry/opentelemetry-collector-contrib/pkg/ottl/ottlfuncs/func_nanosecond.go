// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type nanosecondArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewNanosecondFactory returns a factory for the Nanosecond OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#nanosecond
func NewNanosecondFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Nanosecond", &nanosecondArguments[K]{}, createNanosecondFunction[K])
}

func createNanosecondFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*nanosecondArguments[K])

	if !ok {
		return nil, errors.New("NanosecondFactory args must be of type *nanosecondArguments[K]")
	}

	return nanosecond(args.Time), nil
}

func nanosecond[K any](time ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Nanosecond()), nil
	}
}
