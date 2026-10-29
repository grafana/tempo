// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type containsValueArguments[K any] struct {
	Target ottl.PSliceGetter[K]
	Item   ottl.Getter[K]
}

// NewContainsValueFactory returns a factory for the ContainsValue OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#containsvalue
func NewContainsValueFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ContainsValue", &containsValueArguments[K]{}, createContainsValueFunction[K])
}

func createContainsValueFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*containsValueArguments[K])

	if !ok {
		return nil, errors.New("ContainsValueFactory args must be of type *containsValueArguments[K]")
	}

	return containsValue(args.Target, args.Item), nil
}

func containsValue[K any](target ottl.PSliceGetter[K], itemGetter ottl.Getter[K]) ottl.ExprFunc[K] {
	comparator := ottl.NewValueComparator()

	return func(ctx context.Context, tCtx K) (any, error) {
		slice, sliceErr := target.Get(ctx, tCtx)
		if sliceErr != nil {
			return nil, sliceErr
		}
		item, itemErr := itemGetter.Get(ctx, tCtx)
		if itemErr != nil {
			return nil, itemErr
		}

		n := slice.Len()
		// Compare pcommon.Value in place. We avoid AsRaw() because it copies
		// every element into any, including a full map/slice copy for nested
		// values.
		switch typed := item.(type) {
		case string:
			for i := range n {
				v := slice.At(i)
				if v.Type() == pcommon.ValueTypeStr && v.Str() == typed {
					return true, nil
				}
			}
		case int64:
			for i := range n {
				v := slice.At(i)
				switch v.Type() {
				case pcommon.ValueTypeInt:
					if v.Int() == typed {
						return true, nil
					}
				case pcommon.ValueTypeDouble:
					if v.Double() == float64(typed) {
						return true, nil
					}
				}
			}
		case float64:
			for i := range n {
				v := slice.At(i)
				switch v.Type() {
				case pcommon.ValueTypeDouble:
					if v.Double() == typed {
						return true, nil
					}
				case pcommon.ValueTypeInt:
					if float64(v.Int()) == typed {
						return true, nil
					}
				}
			}
		case bool:
			for i := range n {
				v := slice.At(i)
				if v.Type() == pcommon.ValueTypeBool && v.Bool() == typed {
					return true, nil
				}
			}
		default:
			for i := range n {
				if comparator.Equal(slice.At(i), item) {
					return true, nil
				}
			}
		}
		return false, nil
	}
}
