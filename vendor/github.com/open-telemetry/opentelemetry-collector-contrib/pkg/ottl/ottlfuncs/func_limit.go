// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type LimitArguments[K any] struct {
	Target       ottl.PMapGetter[K]
	Limit        int64
	PriorityKeys []string
}

func NewLimitFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("limit", &LimitArguments[K]{}, createLimitFunction[K])
}

func createLimitFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*LimitArguments[K])

	if !ok {
		return nil, fmt.Errorf("LimitFactory args must be of type *LimitArguments[K]")
	}

	return limit(args.Target, args.Limit, args.PriorityKeys)
}

func limit[K any](target ottl.PMapGetter[K], limit int64, priorityKeys []string) (ottl.ExprFunc[K], error) {
	if limit < 0 {
		return nil, fmt.Errorf("invalid limit for limit function, %d cannot be negative", limit)
	}
	if limit < int64(len(priorityKeys)) {
		return nil, fmt.Errorf(
			"invalid limit for limit function, %d cannot be less than number of priority attributes %d",
			limit, len(priorityKeys),
		)
	}
	keep := make(map[string]struct{}, len(priorityKeys))
	for _, key := range priorityKeys {
		keep[key] = struct{}{}
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		if int64(val.Len()) <= limit {
			return nil, nil
		}

		count := int64(0)
		for _, key := range priorityKeys {
			if _, ok := val.Get(key); ok {
				count++
			}
		}

		val.RemoveIf(func(key string, value pcommon.Value) bool {
			if _, ok := keep[key]; ok {
				return false
			}
			if count < limit {
				count++
				return false
			}
			return true
		})
		// TODO: Write log when limiting is performed
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/9730
		return nil, nil
	}, nil
}
