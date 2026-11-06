// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/slicegetter"
)

type keepKeysArguments[K any] struct {
	Target ottl.PMapGetSetter[K]
	Keys   slicegetter.SliceGetter[K, ottl.StringGetter[K]]
}

// NewKeepKeysFactory returns a factory for the keep_keys OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#keep_keys
func NewKeepKeysFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("keep_keys", &keepKeysArguments[K]{}, createKeepKeysFunction[K])
}

func createKeepKeysFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*keepKeysArguments[K])

	if !ok {
		return nil, errors.New("KeepKeysFactory args must be of type *keepKeysArguments[K]")
	}

	return keepKeys(args.Target, &args.Keys)
}

func keepKeys[K any](target ottl.PMapGetSetter[K], keys *slicegetter.SliceGetter[K, ottl.StringGetter[K]]) (ottl.ExprFunc[K], error) {
	var literalKeySet map[string]struct{}
	keySetCapacity, _ := keys.Len()
	if literalValues, allLiteral := slicegetter.GetLiteralValues(keys, func(key ottl.StringGetter[K]) (string, bool) {
		return ottl.GetLiteralValue[K, string](key)
	}); allLiteral {
		if literalValues == nil {
			return nil, errors.New("keys cannot be nil")
		}

		literalKeySet = make(map[string]struct{}, len(literalValues))
		for _, key := range literalValues {
			literalKeySet[key] = struct{}{}
		}
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		keySet := literalKeySet
		if keySet == nil {
			keySet = make(map[string]struct{}, keySetCapacity)

			var keyErr error
			nonNil, err := keys.Range(ctx, tCtx, func(key ottl.StringGetter[K]) bool {
				k, err := key.Get(ctx, tCtx)
				if err != nil {
					keyErr = err
					return false
				}
				keySet[k] = struct{}{}
				return true
			})
			if err != nil {
				return nil, err
			}
			if keyErr != nil {
				return nil, keyErr
			}
			if !nonNil {
				return nil, errors.New("keys cannot be nil")
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
	}, nil
}
