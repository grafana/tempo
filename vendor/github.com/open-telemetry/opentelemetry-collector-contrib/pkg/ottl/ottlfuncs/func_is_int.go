// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsIntArguments[K any] struct {
	Target ottl.IntGetter[K]
}

func NewIsIntFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsInt", &IsIntArguments[K]{}, createIsIntFunction[K])
}

func createIsIntFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsIntArguments[K])

	if !ok {
		return nil, errors.New("IsIntFactory args must be of type *IsIntArguments[K]")
	}

	return isInt(args.Target), nil
}

//nolint:errorlint
func isInt[K any](target ottl.IntGetter[K]) ottl.ExprFunc[K] {
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
