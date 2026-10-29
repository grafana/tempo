// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var _ reflectTypedArg = (*SliceGetter[any, any])(nil)

// SliceGetter represents a slice argument in OTTL functions. Unlike bare []V
// parameters, which only accept literal lists [value], it also accepts a single
// getter (path or expression) that resolves to []V at runtime.
// V is the element type of the resolved slice. It may be a typed Getter
// (e.g.: StringGetter[K]) or a scalar type supported by OTTL.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
type SliceGetter[K, V any] struct {
	typedValues  []V
	runtimeSlice *runtimeSliceSource[K]
}

func (*SliceGetter[K, V]) addrReflectValue() any {
	return nil
}

func (*SliceGetter[K, V]) reflectTypeParam() reflect.Type {
	return reflect.TypeFor[V]()
}

func (s *SliceGetter[K, V]) setReflectValue(val reflect.Value) error {
	switch v := val.Interface().(type) {
	case []V:
		s.typedValues = v
	case runtimeSliceSource[K]:
		if typedValues, ok := getRuntimeSliceLiterals[K, V](&v); ok {
			s.typedValues = typedValues
		} else {
			s.runtimeSlice = &v
		}
	default:
		return TypeError(fmt.Sprintf("cannot set value of type %s to a slice of type %s", val.Type(), reflect.TypeFor[V]()))
	}
	return nil
}

// getRuntimeSliceLiterals extracts slice literals from a runtimeSliceSource. It returns the
// slice and a boolean indicating if the extraction was successful. The returned values can be
// either scalar or typed getters, which might not hold literal values. In this context, literals
// mean items can be retrieved from the slice without evaluating it.
func getRuntimeSliceLiterals[K, V any](slice *runtimeSliceSource[K]) ([]V, bool) {
	if !isLiteralGetter(slice.Getter) {
		return nil, false
	}
	sliceValues, err := slice.Get(context.Background(), *new(K))
	if err != nil {
		return nil, false
	}
	if sliceValues == nil {
		return nil, true
	}
	if typedValues, ok := sliceValues.([]V); ok {
		return typedValues, true
	}

	var result []V
	if size, ok := slice.sliceLen(sliceValues); ok {
		result = make([]V, 0, size)
	}
	complete := true
	err = slice.rangeSlice(sliceValues, func(val any) bool {
		if tv, ok := val.(V); ok {
			result = append(result, tv)
			return true
		}
		complete = false
		return false
	})
	if err != nil {
		return nil, false
	}
	if !complete {
		return nil, false
	}
	return result, complete
}

// GetScalarLiteralValues retrieves the literal values from the given slice of scalars.
// If the values cannot be retrieved, it returns the zero value of []V and false.
// [V] must be a scalar type supported by OTTL slice arguments.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func GetScalarLiteralValues[
	K any,
	V ~uint8 | ~int64 | ~float64 | ~string, // same as buildSliceArg
](slice *SliceGetter[K, V]) ([]V, bool) {
	var result []V
	err := slice.Range(
		context.Background(),
		*new(K),
		func(value V) bool {
			result = append(result, value)
			return true
		},
	)
	if err != nil {
		return nil, false
	}
	return result, true
}

// GetLiteralValues retrieves the literal values from the given slice of getters.
// If the getter or the value it's currently holding is not a literal value, it
// returns the zero value of []V and false.
// [G] is the type of the slice elements, which must be typed Getter.
// [V] is the expected type of the slice values.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func GetLiteralValues[K, V any, G typedGetter[K, V]](slice *SliceGetter[K, G]) ([]V, bool) {
	var result []V
	allLiterals := true
	err := slice.Range(context.Background(), *new(K), func(value G) bool {
		val, ok := GetLiteralValue(value)
		if !ok {
			allLiterals = false
			return false
		}
		result = append(result, val)
		return true
	})
	if err != nil {
		return nil, false
	}
	if !allLiterals {
		return nil, false
	}
	return result, true
}

// Range iterates over elements in the slice and applies the yield function to each item.
// Return true to continue iterating, false to stop.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func (s *SliceGetter[K, V]) Range(ctx context.Context, tCtx K, yield func(value V) bool) error {
	if s.typedValues != nil {
		return rangeTypedSlice(s.typedValues, yield)
	}
	return s.rangeRuntimeSlice(ctx, tCtx, yield)
}

func (s *SliceGetter[K, V]) rangeRuntimeSlice(ctx context.Context, tCtx K, yield func(value V) bool) error {
	values, err := s.runtimeSlice.Get(ctx, tCtx)
	if err != nil {
		return err
	}
	if values == nil {
		return nil
	}

	if typedValues, ok := values.([]V); ok {
		return rangeTypedSlice(typedValues, yield)
	}

	var rangeErr error
	err = s.runtimeSlice.rangeSlice(values, func(val any) bool {
		if v, ok := val.(V); ok {
			return yield(v)
		}
		rangeErr = TypeError(fmt.Sprintf("expected slice item of type %s, got %s", reflect.TypeFor[V](), reflect.TypeOf(val)))
		return false
	})
	if err != nil {
		return err
	}

	return rangeErr
}

func rangeTypedSlice[V any](typedValues []V, yield func(value V) bool) error {
	for _, v := range typedValues {
		if !yield(v) {
			return nil
		}
	}
	return nil
}

// Len returns the length of the slice when it can be determined without evaluation.
// For literal slices it returns the length and true, otherwise it returns 0 and false.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func (s *SliceGetter[K, V]) Len() (int, bool) {
	if s.typedValues != nil {
		return len(s.typedValues), true
	}
	return 0, false
}

// Get retrieves all values as []V.
// If any slice element is not coercible to V, it returns an error.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func (s *SliceGetter[K, V]) Get(ctx context.Context, tCtx K) ([]V, error) {
	if s.typedValues != nil {
		return s.typedValues, nil
	}

	values, err := s.runtimeSlice.Get(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	if values == nil {
		return nil, nil
	}

	if typedValues, ok := values.([]V); ok {
		return typedValues, nil
	}

	var result []V
	if count, ok := s.runtimeSlice.sliceLen(values); ok {
		result = make([]V, 0, count)
	}

	var rangeErr error
	err = s.runtimeSlice.rangeSlice(values, func(val any) bool {
		if v, ok := val.(V); ok {
			result = append(result, v)
			return true
		}
		rangeErr = TypeError(fmt.Sprintf("expected slice item of type %s, got %s", reflect.TypeFor[V](), reflect.TypeOf(val)))
		return false
	})
	if err != nil {
		return nil, err
	}

	if rangeErr != nil {
		return nil, rangeErr
	}

	return result, nil
}

// sliceElementCoercer is a generic type for handling SliceGetter element coercion operations.
// It stores metadata and provides functionality to coerce and iterate over slice elements.
type sliceElementCoercer[K any] struct {
	sliceItemType        reflect.Type
	sliceItemTypeName    string
	buildSliceItemGetter func(string, Getter[K]) (any, error)
}

func newSliceElementCoercer[K any](
	sliceItemType reflect.Type,
	buildSliceItemGetter func(string, Getter[K]) (any, error),
) *sliceElementCoercer[K] {
	return &sliceElementCoercer[K]{
		sliceItemType:        sliceItemType,
		sliceItemTypeName:    sliceItemType.Name(),
		buildSliceItemGetter: buildSliceItemGetter,
	}
}

func isLiteralSliceElementType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Uint8, reflect.Float64, reflect.Int64:
		return true
	default:
		return false
	}
}

type runtimeSliceSource[K any] struct {
	Getter[K]
	*sliceElementCoercer[K]
}

func buildSliceGetterValue[K any](
	val value,
	sliceItemType reflect.Type,
	buildSliceArg func(value, reflect.Type) (any, error),
	buildSliceItemGetter func(string, Getter[K]) (any, error),
	buildGetter func(value) (Getter[K], error),
) (any, error) {
	if val.List != nil || isLiteralSliceElementType(sliceItemType) {
		return buildSliceArg(val, reflect.SliceOf(sliceItemType))
	}

	valueGetter, err := buildGetter(val)
	if err != nil {
		return nil, err
	}

	return runtimeSliceSource[K]{
		Getter:              valueGetter,
		sliceElementCoercer: newSliceElementCoercer(sliceItemType, buildSliceItemGetter),
	}, nil
}

// sliceLen returns the length of a slice and a boolean indicating if it is a valid slice.
// If the value is not a slice, it returns 0 and false.
func (*sliceElementCoercer[K]) sliceLen(slice any) (int, bool) {
	switch typedVal := slice.(type) {
	case pcommon.Slice:
		return typedVal.Len(), true
	case pcommon.Value:
		if typedVal.Type() != pcommon.ValueTypeSlice {
			return 0, false
		}
		return typedVal.Slice().Len(), true
	default:
		values := reflect.ValueOf(slice)
		if values.Kind() != reflect.Slice {
			return 0, false
		}
		return values.Len(), true
	}
}

// rangeSlice iterates over the slice and applies the yield function to each item after
// coercing the item to the slice item type. Yield true to continue iterating, false to stop.
func (c *sliceElementCoercer[K]) rangeSlice(slice any, yield func(val any) bool) error {
	switch typedVal := slice.(type) {
	case pcommon.Slice:
		for _, item := range typedVal.All() {
			itemGetter, err := c.buildSliceItemGetter(c.sliceItemTypeName, newLiteral[K, any](item))
			if err != nil {
				return err
			}
			if !yield(itemGetter) {
				return nil
			}
		}
	case pcommon.Value:
		if typedVal.Type() != pcommon.ValueTypeSlice {
			return fmt.Errorf("expected a slice, got %q", typedVal.Type())
		}
		return c.rangeSlice(typedVal.Slice(), yield)
	default:
		values := reflect.ValueOf(slice)
		if values.Kind() != reflect.Slice {
			return fmt.Errorf("expected a slice, got %T", slice)
		}
		for i := 0; i < values.Len(); i++ {
			item := values.Index(i)
			if item.Type() == c.sliceItemType {
				if !yield(item.Interface()) {
					return nil
				}
			} else {
				var itemGetter any
				var err error
				rawValue := values.Index(i).Interface()
				if getter, ok := rawValue.(Getter[K]); ok {
					itemGetter, err = c.buildSliceItemGetter(c.sliceItemTypeName, getter)
				} else {
					itemGetter, err = c.buildSliceItemGetter(c.sliceItemTypeName, newLiteral[K, any](rawValue))
				}
				if err != nil {
					return err
				}
				if !yield(itemGetter) {
					return nil
				}
			}
		}
	}
	return nil
}

// NewTestingSliceGetter creates a SliceGetter that resolves a slice at runtime or uses literals.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func NewTestingSliceGetter[K, T any](literal bool, values []T) *SliceGetter[K, T] {
	if literal {
		return &SliceGetter[K, T]{typedValues: values}
	}

	pc := parseContext[K]{}
	sliceItemType := reflect.TypeFor[T]()
	return &SliceGetter[K, T]{
		runtimeSlice: &runtimeSliceSource[K]{
			Getter:              newLiteral[K, any](values),
			sliceElementCoercer: newSliceElementCoercer[K](sliceItemType, pc.buildStandardGetSetter),
		},
	}
}
