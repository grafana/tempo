// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ContainsValueArguments[K any] struct {
	Target ottl.PSliceGetter[K]
	Item   ottl.Getter[K]
}

func NewContainsValueFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ContainsValue", &ContainsValueArguments[K]{}, createContainsValueFunction[K])
}

func createContainsValueFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ContainsValueArguments[K])

	if !ok {
		return nil, errors.New("ContainsValueFactory args must be of type *ContainsValueArguments[K]")
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

		for i := 0; i < slice.Len(); i++ {
			val := slice.At(i).AsRaw()
			if comparator.Equal(val, item) {
				return true, nil
			}
		}
		return false, nil
	}
}
