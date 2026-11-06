// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/iancoleman/strcase"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type toSnakeCaseArguments[K any] struct {
	Target ottl.StringGetter[K]
}

// NewToSnakeCaseFactory returns a factory for the ToSnakeCase OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#tosnakecase
func NewToSnakeCaseFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ToSnakeCase", &toSnakeCaseArguments[K]{}, createToSnakeCaseFunction[K])
}

func createToSnakeCaseFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*toSnakeCaseArguments[K])

	if !ok {
		return nil, errors.New("ToSnakeCaseFactory args must be of type *toSnakeCaseArguments[K]")
	}

	return toSnakeCase(args.Target), nil
}

func toSnakeCase[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if val == "" {
			return val, nil
		}

		return strcase.ToSnake(val), nil
	}
}
