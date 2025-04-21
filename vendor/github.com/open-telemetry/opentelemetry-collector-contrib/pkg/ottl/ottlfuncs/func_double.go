// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type DoubleArguments[K any] struct {
	Target ottl.FloatLikeGetter[K]
}

func NewDoubleFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Double", &DoubleArguments[K]{}, createDoubleFunction[K])
}

func createDoubleFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*DoubleArguments[K])

	if !ok {
		return nil, errors.New("DoubleFactory args must be of type *DoubleArguments[K]")
	}

	return doubleFunc(args.Target), nil
}

func doubleFunc[K any](target ottl.FloatLikeGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		value, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		return *value, nil
	}
}
