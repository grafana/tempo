// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type NanosecondArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewNanosecondFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Nanosecond", &NanosecondArguments[K]{}, createNanosecondFunction[K])
}

func createNanosecondFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*NanosecondArguments[K])

	if !ok {
		return nil, errors.New("NanosecondFactory args must be of type *NanosecondArguments[K]")
	}

	return Nanosecond(args.Time)
}

func Nanosecond[K any](time ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := time.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return int64(t.Nanosecond()), nil
	}, nil
}
