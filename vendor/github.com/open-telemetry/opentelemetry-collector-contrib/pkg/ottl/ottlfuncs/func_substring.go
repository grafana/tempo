// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SubstringArguments[K any] struct {
	Target ottl.StringGetter[K]
	Start  ottl.IntGetter[K]
	Length ottl.IntGetter[K]
}

func NewSubstringFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Substring", &SubstringArguments[K]{}, createSubstringFunction[K])
}

func createSubstringFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SubstringArguments[K])

	if !ok {
		return nil, errors.New("SubstringFactory args must be of type *SubstringArguments[K]")
	}

	return substring(args.Target, args.Start, args.Length), nil
}

func substring[K any](target ottl.StringGetter[K], startGetter ottl.IntGetter[K], lengthGetter ottl.IntGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		start, err := startGetter.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if start < 0 {
			return nil, fmt.Errorf("invalid start for substring function, %d cannot be negative", start)
		}
		length, err := lengthGetter.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if length <= 0 {
			return nil, fmt.Errorf("invalid length for substring function, %d cannot be negative or zero", length)
		}
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		if (start + length) > int64(len(val)) {
			return nil, fmt.Errorf("invalid range for substring function, %d cannot be greater than the length of target string(%d)", start+length, len(val))
		}
		return val[start : start+length], nil
	}
}
