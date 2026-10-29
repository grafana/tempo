// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type DeleteIndexArguments[K any] struct {
	Target     ottl.PSliceGetSetter[K]
	StartIndex ottl.IntGetter[K]
	EndIndex   ottl.Optional[ottl.IntGetter[K]]
}

func NewDeleteIndexFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("delete_index", &DeleteIndexArguments[K]{}, createDeleteIndexFunction[K])
}

func createDeleteIndexFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*DeleteIndexArguments[K])
	if !ok {
		return nil, errors.New("DeleteIndexFactory args must be of type *DeleteIndexArguments[K]")
	}

	return deleteIndexFrom(args.Target, args.StartIndex, args.EndIndex), nil
}

func deleteIndexFrom[K any](target ottl.PSliceGetSetter[K], startIndexGetter ottl.IntGetter[K], endIndexGetter ottl.Optional[ottl.IntGetter[K]]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		t, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		startIndex, err := startIndexGetter.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		sliceLen := int64(t.Len())
		endIndex := startIndex + 1
		if !endIndexGetter.IsEmpty() {
			endIndex, err = endIndexGetter.Get().Get(ctx, tCtx)
			if err != nil {
				return nil, err
			}
		}

		if startIndex == 0 && endIndex == sliceLen {
			// If deleting all elements, return an empty slice without looping
			return nil, target.Set(ctx, tCtx, pcommon.NewSlice())
		}

		err = validateBounds(startIndex, endIndex, sliceLen)
		if err != nil {
			return nil, err
		}

		if startIndex == endIndex {
			// No elements to delete
			return nil, target.Set(ctx, tCtx, t)
		}

		var i int64
		t.RemoveIf(func(_ pcommon.Value) bool {
			remove := i >= startIndex && i < endIndex
			i++
			return remove
		})
		return nil, target.Set(ctx, tCtx, t)
	}
}

func validateBounds(startIndex, endIndex, sliceLen int64) error {
	if startIndex < 0 || startIndex >= sliceLen {
		return fmt.Errorf("startIndex %d out of bounds for slice of length %d", startIndex, sliceLen)
	}

	if endIndex < startIndex {
		return fmt.Errorf("endIndex %d cannot be less than startIndex %d", endIndex, startIndex)
	}

	if endIndex > sliceLen {
		return fmt.Errorf("deletion range [%d:%d] out of bounds for slice of length %d", startIndex, endIndex, sliceLen)
	}
	return nil
}
