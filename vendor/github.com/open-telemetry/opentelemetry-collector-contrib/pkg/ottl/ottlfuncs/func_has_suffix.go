// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type HasSuffixArguments[K any] struct {
	Target ottl.StringGetter[K]
	Suffix string
}

func NewHasSuffixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("HasSuffix", &HasSuffixArguments[K]{}, createHasSuffixFunction[K])
}

func createHasSuffixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*HasSuffixArguments[K])

	if !ok {
		return nil, errors.New("HasSuffixFactory args must be of type *HasSuffixArguments[K]")
	}

	return HasSuffix(args.Target, args.Suffix)
}

func HasSuffix[K any](target ottl.StringGetter[K], suffix string) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.HasSuffix(val, suffix), nil
	}, nil
}
