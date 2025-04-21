// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ConcatArguments[K any] struct {
	Vals      []ottl.StringLikeGetter[K]
	Delimiter string
}

func NewConcatFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Concat", &ConcatArguments[K]{}, createConcatFunction[K])
}

func createConcatFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ConcatArguments[K])

	if !ok {
		return nil, errors.New("ConcatFactory args must be of type *ConcatArguments[K]")
	}

	return concat(args.Vals, args.Delimiter), nil
}

func concat[K any](vals []ottl.StringLikeGetter[K], delimiter string) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		builder := strings.Builder{}
		for i, rv := range vals {
			val, err := rv.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
			if val == nil {
				builder.WriteString(fmt.Sprint(val))
			} else {
				builder.WriteString(*val)
			}
			if i != len(vals)-1 {
				builder.WriteString(delimiter)
			}
		}
		return builder.String(), nil
	}
}
