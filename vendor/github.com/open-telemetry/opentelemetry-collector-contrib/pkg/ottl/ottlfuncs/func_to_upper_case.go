// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ToUpperCaseArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewToUpperCaseFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ToUpperCase", &ToUpperCaseArguments[K]{}, createToUpperCaseFunction[K])
}

func createToUpperCaseFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ToUpperCaseArguments[K])

	if !ok {
		return nil, errors.New("ToUpperCaseFactory args must be of type *ToUpperCaseArguments[K]")
	}

	return toUpperCase(args.Target), nil
}

func toUpperCase[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if val == "" {
			return val, nil
		}

		return strings.ToUpper(val), nil
	}
}
