// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsMapArguments[K any] struct {
	Target ottl.PMapGetter[K]
}

func NewIsMapFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsMap", &IsMapArguments[K]{}, createIsMapFunction[K])
}

func createIsMapFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsMapArguments[K])

	if !ok {
		return nil, fmt.Errorf("IsMapFactory args must be of type *IsMapArguments[K]")
	}

	return isMap(args.Target), nil
}

// nolint:errorlint
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
