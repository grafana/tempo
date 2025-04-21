// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"golang.org/x/net/context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type SliceToMapArguments[K any] struct {
	Target    ottl.Getter[K]
	KeyPath   []string
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

	return getSliceToMapFunc(args.Target, args.KeyPath, args.ValuePath)
}

func getSliceToMapFunc[K any](target ottl.Getter[K], keyPath []string, valuePath ottl.Optional[[]string]) (ottl.ExprFunc[K], error) {
	if len(keyPath) == 0 {
		return nil, errors.New("key path must contain at least one element")
	}
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		switch v := val.(type) {
		case []any:
			return sliceToMap(v, keyPath, valuePath)
		case pcommon.Slice:
			return sliceToMap(v.AsRaw(), keyPath, valuePath)
		default:
			return nil, fmt.Errorf("unsupported type provided to SliceToMap function: %T", v)
		}
	}, nil
}

func sliceToMap(v []any, keyPath []string, valuePath ottl.Optional[[]string]) (any, error) {
	result := make(map[string]any, len(v))
	for _, elem := range v {
		e, ok := elem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("could not cast element '%v' to map[string]any", elem)
		}
		extractedKey, err := extractValue(e, keyPath)
		if err != nil {
			return nil, fmt.Errorf("could not extract key from element: %w", err)
		}

		key, ok := extractedKey.(string)
		if !ok {
			return nil, errors.New("extracted key attribute is not of type string")
		}

		if valuePath.IsEmpty() {
			result[key] = e
			continue
		}
		extractedValue, err := extractValue(e, valuePath.Get())
		if err != nil {
			return nil, fmt.Errorf("could not extract value from element: %w", err)
		}
		result[key] = extractedValue
	}
	m := pcommon.NewMap()
	if err := m.FromRaw(result); err != nil {
		return nil, fmt.Errorf("could not create pcommon.Map from result: %w", err)
	}

	return m, nil
}

func extractValue(v map[string]any, path []string) (any, error) {
	if len(path) == 0 {
		return nil, errors.New("must provide at least one path item")
	}
	obj, ok := v[path[0]]
	if !ok {
		return nil, fmt.Errorf("provided object does not contain the path %v", path)
	}
	if len(path) == 1 {
		return obj, nil
	}

	if o, ok := obj.(map[string]any); ok {
		return extractValue(o, path[1:])
	}
	return nil, fmt.Errorf("provided object does not contain the path %v", path)
}
