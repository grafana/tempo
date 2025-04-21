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
	INSERT = "insert"
	UPDATE = "update"
	UPSERT = "upsert"
)

type MergeMapsArguments[K any] struct {
	Target   ottl.PMapGetter[K]
	Source   ottl.PMapGetter[K]
	Strategy string
}

func NewMergeMapsFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("merge_maps", &MergeMapsArguments[K]{}, createMergeMapsFunction[K])
}

func createMergeMapsFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*MergeMapsArguments[K])

	if !ok {
		return nil, errors.New("MergeMapsFactory args must be of type *MergeMapsArguments[K]")
	}

	return mergeMaps(args.Target, args.Source, args.Strategy)
}

// mergeMaps function merges the source map into the target map using the supplied strategy to handle conflicts.
// Strategy definitions:
//
//	insert: Insert the value from `source` into `target` where the key does not already exist.
//	update: Update the entry in `target` with the value from `source` where the key does exist
//	upsert: Performs insert or update. Insert the value from `source` into `target` where the key does not already exist and update the entry in `target` with the value from `source` where the key does exist.
func mergeMaps[K any](target ottl.PMapGetter[K], source ottl.PMapGetter[K], strategy string) (ottl.ExprFunc[K], error) {
	if strategy != INSERT && strategy != UPDATE && strategy != UPSERT {
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
		case INSERT:
			for k, v := range valueMap.All() {
				if _, ok := targetMap.Get(k); !ok {
					tv := targetMap.PutEmpty(k)
					v.CopyTo(tv)
				}
			}
		case UPDATE:
			for k, v := range valueMap.All() {
				if tv, ok := targetMap.Get(k); ok {
					v.CopyTo(tv)
				}
			}
		case UPSERT:
			for k, v := range valueMap.All() {
				tv := targetMap.PutEmpty(k)
				v.CopyTo(tv)
			}
		default:
			return nil, fmt.Errorf("unknown strategy, %v", strategy)
		}
		return nil, nil
	}, nil
}
