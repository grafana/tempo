// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type HasPrefixArguments[K any] struct {
	Target ottl.StringGetter[K]
	Prefix string
}

func NewHasPrefixFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("HasPrefix", &HasPrefixArguments[K]{}, createHasPrefixFunction[K])
}

func createHasPrefixFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*HasPrefixArguments[K])

	if !ok {
		return nil, errors.New("HasPrefixFactory args must be of type *HasPrefixArguments[K]")
	}

	return HasPrefix(args.Target, args.Prefix)
}

func HasPrefix[K any](target ottl.StringGetter[K], prefix string) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return strings.HasPrefix(val, prefix), nil
	}, nil
}
