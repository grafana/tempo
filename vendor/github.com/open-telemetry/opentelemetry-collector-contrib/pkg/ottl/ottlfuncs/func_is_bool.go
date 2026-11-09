// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type isBoolArguments[K any] struct {
	Target ottl.BoolGetter[K]
}

// NewIsBoolFactory returns a factory for the IsBool OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#isbool
func NewIsBoolFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsBool", &isBoolArguments[K]{}, createIsBoolFunction[K])
}

func createIsBoolFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*isBoolArguments[K])

	if !ok {
		return nil, errors.New("IsBoolFactory args must be of type *isBoolArguments[K]")
	}

	return isBool(args.Target), nil
}

//nolint:errorlint
func isBool[K any](target ottl.BoolGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		_, err := target.Get(ctx, tCtx)
		// Use type assertion, because we don't want to check wrapped errors
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
