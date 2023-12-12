// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsStringArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewIsStringFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsString", &IsStringArguments[K]{}, createIsStringFunction[K])
}

func createIsStringFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsStringArguments[K])

	if !ok {
		return nil, fmt.Errorf("IsStringFactory args must be of type *IsStringArguments[K]")
	}

	return isString(args.Target), nil
}

// nolint:errorlint
func isString[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		_, err := target.Get(ctx, tCtx)
		// Use type assertion because we don't want to check wrapped errors
		switch err.(type) {
		case ottl.TypeError:
			return false, nil
		case nil:
			return true, nil
		default:
			return false, err
		}
	}
}
