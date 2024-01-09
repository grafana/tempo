// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type UnixMilliArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

func NewUnixMilliFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("UnixMilli", &UnixMilliArguments[K]{}, createUnixMilliFunction[K])
}
func createUnixMilliFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*UnixMilliArguments[K])

	if !ok {
		return nil, fmt.Errorf("UnixMilliFactory args must be of type *UnixMilliArguments[K]")
	}

	return UnixMilli(args.Time)
}

func UnixMilli[K any](inputTime ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return t.UnixMilli(), nil
	}, nil
}
