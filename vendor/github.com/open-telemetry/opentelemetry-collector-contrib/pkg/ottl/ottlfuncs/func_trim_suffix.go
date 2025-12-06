// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TrimSuffixArguments[K any] struct {
	Target ottl.StringGetter[K]
	Suffix ottl.StringGetter[K]
}

func NewTrimSuffixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Trim", &TrimSuffixArguments[K]{}, createTrimSuffixFunction[K])
}

func createTrimSuffixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TrimSuffixArguments[K])

	if !ok {
		return nil, errors.New("TrimFactory args must be of type *TrimSuffixArguments[K]")
	}

	return trimSuffix(args.Target, args.Suffix), nil
}

func trimSuffix[K any](target, prefix ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		prefixVal, err := prefix.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.TrimSuffix(val, prefixVal), nil
	}
}
