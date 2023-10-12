// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type TruncateAllArguments[K any] struct {
	Target ottl.PMapGetter[K] `ottlarg:"0"`
	Limit  int64              `ottlarg:"1"`
}

func NewTruncateAllFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("truncate_all", &TruncateAllArguments[K]{}, createTruncateAllFunction[K])
}

func createTruncateAllFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*TruncateAllArguments[K])

	if !ok {
		return nil, fmt.Errorf("TruncateAllFactory args must be of type *TruncateAllArguments[K]")
	}

	return TruncateAll(args.Target, args.Limit)
}

func TruncateAll[K any](target ottl.PMapGetter[K], limit int64) (ottl.ExprFunc[K], error) {
	if limit < 0 {
		return nil, fmt.Errorf("invalid limit for truncate_all function, %d cannot be negative", limit)
	}
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		if limit < 0 {
			return nil, nil
		}

		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		val.Range(func(key string, value pcommon.Value) bool {
			stringVal := value.Str()
			if int64(len(stringVal)) > limit {
				value.SetStr(stringVal[:limit])
			}
			return true
		})
		// TODO: Write log when truncation is performed
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/9730
		return nil, nil
	}, nil
}
