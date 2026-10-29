// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/iancoleman/strcase"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ToCamelCaseArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewToCamelCaseFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ToCamelCase", &ToCamelCaseArguments[K]{}, createToCamelCaseFunction[K])
}

func createToCamelCaseFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ToCamelCaseArguments[K])

	if !ok {
		return nil, errors.New("ToCamelCaseFactory args must be of type *ToCamelCaseArguments[K]")
	}

	return toCamelCase(args.Target), nil
}

func toCamelCase[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if val == "" {
			return val, nil
		}

		return strcase.ToCamel(val), nil
	}
}
