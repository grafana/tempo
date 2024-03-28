// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsDoubleArguments[K any] struct {
	Target ottl.FloatGetter[K]
}

func NewIsDoubleFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsDouble", &IsDoubleArguments[K]{}, createIsDoubleFunction[K])
}

func createIsDoubleFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsDoubleArguments[K])

	if !ok {
		return nil, fmt.Errorf("IsDoubleFactory args must be of type *IsDoubleArguments[K]")
	}

	return isDouble(args.Target), nil
}

// nolint:errorlint
func isDouble[K any](target ottl.FloatGetter[K]) ottl.ExprFunc[K] {
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
