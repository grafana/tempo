// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type splitArguments[K any] struct {
	Target    ottl.StringGetter[K]
	Delimiter ottl.StringGetter[K]
}

// NewSplitFactory returns a factory for the Split OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#split
func NewSplitFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Split", &splitArguments[K]{}, createSplitFunction[K])
}

func createSplitFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*splitArguments[K])

	if !ok {
		return nil, errors.New("SplitFactory args must be of type *splitArguments[K]")
	}

	return split(args.Target, args.Delimiter), nil
}

func split[K any](target, delimiter ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		delimiterVal, err := delimiter.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.Split(val, delimiterVal), nil
	}
}
