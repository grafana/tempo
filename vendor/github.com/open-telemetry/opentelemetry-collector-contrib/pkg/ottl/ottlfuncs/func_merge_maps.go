// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	insert = "insert"
	update = "update"
	upsert = "upsert"
)

type mergeMapsArguments[K any] struct {
	Target   ottl.PMapGetSetter[K]
	Source   ottl.PMapGetter[K]
	Strategy string
}

// NewMergeMapsFactory returns a factory for the merge_maps OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#merge_maps
func NewMergeMapsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("merge_maps", &mergeMapsArguments[K]{}, createMergeMapsFunction[K])
}

func createMergeMapsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*mergeMapsArguments[K])

	if !ok {
		return nil, errors.New("MergeMapsFactory args must be of type *mergeMapsArguments[K]")
	}

	return mergeMaps(args.Target, args.Source, args.Strategy)
}

// mergeMaps function merges the source map into the target map using the supplied strategy to handle conflicts.
// Strategy definitions:
//
//	insert: Insert the value from `source` into `target` where the key does not already exist.
//	update: Update the entry in `target` with the value from `source` where the key does exist
//	upsert: Performs insert or update. Insert the value from `source` into `target` where the key does not already exist and update the entry in `target` with the value from `source` where the key does exist.
func mergeMaps[K any](target ottl.PMapGetSetter[K], source ottl.PMapGetter[K], strategy string) (ottl.ExprFunc[K], error) {
	if strategy != insert && strategy != update && strategy != upsert {
		return nil, fmt.Errorf("invalid value for strategy, %v, must be 'insert', 'update' or 'upsert'", strategy)
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		targetMap, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		valueMap, err := source.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		switch strategy {
		case insert:
			for k, v := range valueMap.All() {
				if _, ok := targetMap.Get(k); !ok {
					tv := targetMap.PutEmpty(k)
					v.CopyTo(tv)
				}
			}
		case update:
			for k, v := range valueMap.All() {
				if tv, ok := targetMap.Get(k); ok {
					v.CopyTo(tv)
				}
			}
		case upsert:
			for k, v := range valueMap.All() {
				tv := targetMap.PutEmpty(k)
				v.CopyTo(tv)
			}
		default:
			return nil, fmt.Errorf("unknown strategy, %v", strategy)
		}
		return nil, target.Set(ctx, tCtx, targetMap)
	}, nil
}
