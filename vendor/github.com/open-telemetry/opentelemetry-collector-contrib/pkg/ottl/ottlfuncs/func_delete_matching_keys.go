// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type deleteMatchingKeysArguments[K any] struct {
	Target  ottl.PMapGetSetter[K]
	Pattern ottl.StringGetter[K]
}

// NewDeleteMatchingKeysFactory returns a factory for the delete_matching_keys OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#delete_matching_keys
func NewDeleteMatchingKeysFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("delete_matching_keys", &deleteMatchingKeysArguments[K]{}, createDeleteMatchingKeysFunction[K])
}

func createDeleteMatchingKeysFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*deleteMatchingKeysArguments[K])

	if !ok {
		return nil, errors.New("DeleteMatchingKeysFactory args must be of type *deleteMatchingKeysArguments[K]")
	}

	return deleteMatchingKeys(args.Target, args.Pattern)
}

func deleteMatchingKeys[K any](target ottl.PMapGetSetter[K], pattern ottl.StringGetter[K]) (ottl.ExprFunc[K], error) {
	compiledPattern, err := newDynamicRegex("delete_matching_keys", pattern)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, tCtx K) (any, error) {
		cp, err := compiledPattern.compile(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		val.RemoveIf(func(key string, _ pcommon.Value) bool {
			return cp.MatchString(key)
		})
		return nil, target.Set(ctx, tCtx, val)
	}, nil
}
