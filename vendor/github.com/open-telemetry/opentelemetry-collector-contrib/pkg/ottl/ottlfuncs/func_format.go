// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type FormatArguments[K any] struct {
	Format string
	Vals   []ottl.Getter[K]
}

func NewFormatFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Format", &FormatArguments[K]{}, createFormatFunction[K])
}

func createFormatFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*FormatArguments[K])
	if !ok {
		return nil, errors.New("FormatFactory args must be of type *FormatArguments[K]")
	}

	return format(args.Format, args.Vals), nil
}

func format[K any](formatString string, vals []ottl.Getter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		formatArgs := make([]any, 0, len(vals))
		for _, arg := range vals {
			formatArg, err := arg.Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}

			formatArgs = append(formatArgs, formatArg)
		}

		return fmt.Sprintf(formatString, formatArgs...), nil
	}
}
