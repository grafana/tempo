// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type UnixNanoArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewUnixNanoFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("UnixNano", &UnixNanoArguments[K]{}, createUnixNanoFunction[K])
}
func createUnixNanoFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*UnixNanoArguments[K])

	if !ok {
		return nil, fmt.Errorf("UnixNanoFactory args must be of type *UnixNanoArguments[K]")
	}

	return UnixNano(args.Time)
}

func UnixNano[K any](inputTime ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return t.UnixNano(), nil
	}, nil
}
