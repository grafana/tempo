// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsBoolArguments[K any] struct {
	Target ottl.BoolGetter[K]
}

func NewIsBoolFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsBool", &IsBoolArguments[K]{}, createIsBoolFunction[K])
}

func createIsBoolFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsBoolArguments[K])

	if !ok {
		return nil, errors.New("IsBoolFactory args must be of type *IsBoolArguments[K]")
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
