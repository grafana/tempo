// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type toLowerCaseArguments[K any] struct {
	Target ottl.StringGetter[K]
}

// NewToLowerCaseFactory returns a factory for the ToLowerCase OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#tolowercase
func NewToLowerCaseFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ToLowerCase", &toLowerCaseArguments[K]{}, createToLowerCaseFunction[K])
}

func createToLowerCaseFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*toLowerCaseArguments[K])

	if !ok {
		return nil, errors.New("ToLowerCaseFactory args must be of type *toLowerCaseArguments[K]")
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
