// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type KeepKeysArguments[K any] struct {
	Target ottl.PMapGetSetter[K]
	Keys   []ottl.StringGetter[K]
}

func NewKeepKeysFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("keep_keys", &KeepKeysArguments[K]{}, createKeepKeysFunction[K])
}

func createKeepKeysFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*KeepKeysArguments[K])

	if !ok {
		return nil, errors.New("KeepKeysFactory args must be of type *KeepKeysArguments[K]")
	}

	return keepKeys(args.Target, args.Keys), nil
}

func keepKeys[K any](target ottl.PMapGetSetter[K], keys []ottl.StringGetter[K]) ottl.ExprFunc[K] {
	// Check if all keys are literals and pre-build the key set if so
	literalKeySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		k, isLiteral := ottl.GetLiteralValue(key)
		if !isLiteral {
			literalKeySet = nil
			break
		}
		literalKeySet[k] = struct{}{}
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		var keySet map[string]struct{}
		if literalKeySet != nil {
			// Use pre-built key set for literal keys
			keySet = literalKeySet
		} else {
			// Build key set at runtime for dynamic keys
			keySet = make(map[string]struct{}, len(keys))
			for _, key := range keys {
				k, err := key.Get(ctx, tCtx)
				if err != nil {
					return nil, err
				}
				keySet[k] = struct{}{}
			}
		}

		val.RemoveIf(func(key string, _ pcommon.Value) bool {
			_, ok := keySet[key]
			return !ok
		})
		if val.Len() == 0 {
			val.Clear()
		}
		return nil, target.Set(ctx, tCtx, val)
	}
}
