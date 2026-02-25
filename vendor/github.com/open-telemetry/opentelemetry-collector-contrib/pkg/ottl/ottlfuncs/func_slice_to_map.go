// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SliceToMapArguments[K any] struct {
	Target    ottl.PSliceGetter[K]
	KeyPath   ottl.Optional[[]string]
	ValuePath ottl.Optional[[]string]
}

func NewSliceToMapFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("SliceToMap", &SliceToMapArguments[K]{}, sliceToMapFunction[K])
}

func sliceToMapFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*SliceToMapArguments[K])
	if !ok {
		return nil, errors.New("SliceToMapFactory args must be of type *SliceToMapArguments[K")
	}

	return getSliceToMapFunc(args.Target, args.KeyPath, args.ValuePath), nil
}

func getSliceToMapFunc[K any](target ottl.PSliceGetter[K], keyPath, valuePath ottl.Optional[[]string]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		return sliceToMap(val, keyPath, valuePath)
	}
}

func sliceToMap(v pcommon.Slice, keyPath, valuePath ottl.Optional[[]string]) (pcommon.Map, error) {
	m := pcommon.NewMap()
	m.EnsureCapacity(v.Len())

	useKeyPath := !keyPath.IsEmpty()
	useValuePath := !valuePath.IsEmpty()

	for i, elem := range v.All() {
		var key string
		// If value_path is not set, value is the whole element
		value := elem

		if useKeyPath || useValuePath {
			if elem.Type() != pcommon.ValueTypeMap {
				return pcommon.Map{}, fmt.Errorf("slice elements must be maps when using `key_path` or `value_path`, but could not cast element of type `%s` to a map", elem.Type())
			}
		}

		if useKeyPath {
			extractedKey, err := extractValue(elem.Map(), keyPath.Get())
			if err != nil {
				return pcommon.Map{}, fmt.Errorf("could not extract key from element %d: %w", i, err)
			}
			if extractedKey.Type() != pcommon.ValueTypeStr {
				return pcommon.Map{}, fmt.Errorf("element %d: extracted key attribute is not of type string, got %q", i, extractedKey.Type())
			}
			key = extractedKey.Str()
		} else {
			key = strconv.Itoa(i)
		}

		if useValuePath {
			extractedValue, err := extractValue(elem.Map(), valuePath.Get())
			if err != nil {
				return pcommon.Map{}, fmt.Errorf("could not extract value from element %d: %w", i, err)
			}
			value = extractedValue
		}

		value.CopyTo(m.PutEmpty(key))
	}
	return m, nil
}

func extractValue(m pcommon.Map, path []string) (pcommon.Value, error) {
	if len(path) == 0 {
		return pcommon.Value{}, errors.New("must provide at least one path item")
	}

	val, ok := m.Get(path[0])
	if !ok {
		return pcommon.Value{}, fmt.Errorf("provided object does not contain the path %v", path)
	}

	if len(path) == 1 {
		return val, nil
	}

	if val.Type() != pcommon.ValueTypeMap {
		return pcommon.Value{}, fmt.Errorf("provided object does not contain the path %v", path)
	}

	return extractValue(val.Map(), path[1:])
}
