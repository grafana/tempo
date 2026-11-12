// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type hasSuffixArguments[K any] struct {
	Target ottl.StringGetter[K]
	Suffix ottl.StringGetter[K]
}

// NewHasSuffixFactory returns a factory for the HasSuffix OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#hassuffix
func NewHasSuffixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("HasSuffix", &hasSuffixArguments[K]{}, createHasSuffixFunction[K])
}

func createHasSuffixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*hasSuffixArguments[K])

	if !ok {
		return nil, errors.New("HasSuffixFactory args must be of type *hasSuffixArguments[K]")
	}

	return hasSuffix(args.Target, args.Suffix), nil
}

func hasSuffix[K any](target, suffix ottl.StringGetter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		suffixVal, err := suffix.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.HasSuffix(val, suffixVal), nil
	}
}
