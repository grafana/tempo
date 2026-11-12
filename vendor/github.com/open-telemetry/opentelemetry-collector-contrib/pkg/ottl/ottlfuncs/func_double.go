// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type doubleArguments[K any] struct {
	Target ottl.FloatLikeGetter[K]
}

// NewDoubleFactory returns a factory for the Double OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#double
func NewDoubleFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Double", &doubleArguments[K]{}, createDoubleFunction[K])
}

func createDoubleFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*doubleArguments[K])

	if !ok {
		return nil, errors.New("DoubleFactory args must be of type *doubleArguments[K]")
	}

	return doubleFunc(args.Target), nil
}

func doubleFunc[K any](target ottl.FloatLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, ok, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
		return value, nil
	}
}
