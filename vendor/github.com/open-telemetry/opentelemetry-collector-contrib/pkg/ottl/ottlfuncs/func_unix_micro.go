// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type unixMicroArguments[K any] struct {
	Time ottl.TimeGetter[K]
}

// NewUnixMicroFactory returns a factory for the UnixMicro OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#unixmicro
func NewUnixMicroFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("UnixMicro", &unixMicroArguments[K]{}, createUnixMicroFunction[K])
}

func createUnixMicroFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*unixMicroArguments[K])

	if !ok {
		return nil, errors.New("UnixMicroFactory args must be of type *unixMicroArguments[K]")
	}

	return unixMicro(args.Time), nil
}

func unixMicro[K any](inputTime ottl.TimeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := inputTime.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return t.UnixMicro(), nil
	}
}
