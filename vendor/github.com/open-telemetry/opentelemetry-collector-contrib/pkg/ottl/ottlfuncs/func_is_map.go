// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type isMapArguments[K any] struct {
	Target ottl.PMapGetter[K]
}

// NewIsMapFactory returns a factory for the IsMap OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#ismap
func NewIsMapFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsMap", &isMapArguments[K]{}, createIsMapFunction[K])
}

func createIsMapFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*isMapArguments[K])

	if !ok {
		return nil, errors.New("IsMapFactory args must be of type *isMapArguments[K]")
	}

	return isMap(args.Target), nil
}

//nolint:errorlint
func isMap[K any](target ottl.PMapGetter[K]) ottl.ExprFunc[K] {
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
