// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type parseIntArguments[K any] struct {
	Target ottl.StringGetter[K]
	Base   ottl.IntGetter[K]
}

// NewParseIntFactory returns a factory for the ParseInt OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#parseint
func NewParseIntFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ParseInt", &parseIntArguments[K]{}, createParseIntFunction[K])
}

func createParseIntFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*parseIntArguments[K])

	if !ok {
		return nil, errors.New("ParseIntFactory args must be of type *parseIntArguments[K]")
	}

	return parseIntFunc(args.Target, args.Base), nil
}

func parseIntFunc[K any](target ottl.StringGetter[K], base ottl.IntGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		targetValue, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if targetValue == "" {
			return nil, errors.New("invalid target value for ParseInt function, target cannot be empty")
		}
		baseValue, err := base.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if baseValue < 0 {
			return nil, fmt.Errorf("invalid base value: %d for ParseInt function, base cannot be negative", baseValue)
		}
		result, err := strconv.ParseInt(targetValue, int(baseValue), 64)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}
