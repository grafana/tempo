// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type UnixMicroArguments[K any] struct {
	Time ottl.TimeGetter[K] `ottlarg:"0"`
}

func NewUnixMicroFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("UnixMicro", &UnixMicroArguments[K]{}, createUnixMicroFunction[K])
}
func createUnixMicroFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*UnixMicroArguments[K])

	if !ok {
		return nil, fmt.Errorf("UnixMicroFactory args must be of type *UnixMicroArguments[K]")
	}

	return UnixMicro(args.Time)
}

func UnixMicro[K any](inputTime ottl.TimeGetter[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return t.UnixMicro(), nil
	}, nil
}
