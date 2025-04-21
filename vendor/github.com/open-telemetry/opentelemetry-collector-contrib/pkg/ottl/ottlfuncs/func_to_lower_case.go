// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ToLowerCaseArguments[K any] struct {
	Target ottl.StringGetter[K]
}

func NewToLowerCaseFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ToLowerCase", &ToLowerCaseArguments[K]{}, createToLowerCaseFunction[K])
}

func createToLowerCaseFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ToLowerCaseArguments[K])

	if !ok {
		return nil, errors.New("ToLowerCaseFactory args must be of type *ToLowerCaseArguments[K]")
	}

	return toLowerCase(args.Target), nil
}

func toLowerCase[K any](target ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if val == "" {
			return val, nil
		}

		return strings.ToLower(val), nil
	}
}
