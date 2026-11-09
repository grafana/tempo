// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type yearArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewYearFactory returns a factory for the Year OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#year
func NewYearFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Year", &yearArguments[K]{}, createYearFunction[K])
}

func createYearFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*yearArguments[K])

	if !ok {
		return nil, errors.New("YearFactory args must be of type *yearArguments[K]")
	}

	return year(args.Time), nil
}

func year[K any](time ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Year()), nil
	}
}
