// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/slicegetter"
)

type concatArguments[K any] struct {
	Vals      slicegetter.SliceGetter[K, ottl.StringLikeGetter[K]]
	Delimiter ottl.StringGetter[K]
}

// NewConcatFactory returns a factory for the Concat OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#concat
func NewConcatFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Concat", &concatArguments[K]{}, createConcatFunction[K])
}

func createConcatFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*concatArguments[K])

	if !ok {
		return nil, errors.New("ConcatFactory args must be of type *concatArguments[K]")
	}

	return concat(&args.Vals, args.Delimiter), nil
}

func concat[K any](vals *slicegetter.SliceGetter[K, ottl.StringLikeGetter[K]], delimiter ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		builder := strings.Builder{}
		delimiterVal, err := delimiter.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		first := true
		var concatErr error
		_, err = vals.Range(ctx, tCtx, func(rv ottl.StringLikeGetter[K]) bool {
			val, ok, getErr := rv.Get(ctx, tCtx)
			if getErr != nil {
				concatErr = getErr
				return false
			}
			if !first {
				builder.WriteString(delimiterVal)
			}
			first = false
			if !ok {
				builder.WriteString("<nil>")
			} else {
				builder.WriteString(val)
			}
			return true
		})
		if err != nil {
			return nil, err
		}
		if concatErr != nil {
			return nil, concatErr
		}
		return builder.String(), nil
	}
}
