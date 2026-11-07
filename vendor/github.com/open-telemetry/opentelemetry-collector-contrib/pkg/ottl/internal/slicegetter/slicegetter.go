// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package slicegetter implements the SliceGetter OTTL function argument. It is exposed
// publicly by the xottl module and used internally by stable OTTL functions.
package slicegetter // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/slicegetter"

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Getter mirrors ottl.Getter so this package does not depend on the ottl package.
type Getter[K any] interface {
	Get(ctx context.Context, tCtx K) (any, error)
}

// reflectTypedArg is implemented by every SliceGetter instantiation so the OTTL parser can
// populate one without knowing its type parameters.
type reflectTypedArg interface {
	reflectTypeParam() reflect.Type
	setReflectValue(val reflect.Value) error
}

// ReflectTypeParam returns the element type of the SliceGetter pointed to by ptr, and
// false if ptr is not a pointer to a SliceGetter.
func ReflectTypeParam(ptr any) (reflect.Type, bool) {
	arg, ok := ptr.(reflectTypedArg)
	if !ok {
		return nil, false
	}
	return arg.reflectTypeParam(), true
}

// SetReflectValue stores val, which must hold a []V or a value returned by
// NewRuntimeSliceSource, into the SliceGetter pointed to by ptr.
func SetReflectValue(ptr any, val reflect.Value) error {
	arg, ok := ptr.(reflectTypedArg)
	if !ok {
		return fmt.Errorf("cannot set a slice argument on %T", ptr)
	}
	return arg.setReflectValue(val)
}

var _ reflectTypedArg = (*SliceGetter[any, any])(nil)

// SliceGetter represents a slice argument in OTTL functions. Unlike bare []V
// parameters, which only accept literal lists [value], it also accepts a single
// getter (path or expression) that resolves to []V at runtime.
// V is the element type of the resolved slice. It may be a typed Getter
// (e.g.: StringGetter[K]) or a scalar type supported by OTTL.
type SliceGetter[K, V any] struct {
	typedValues  []V
	runtimeSlice *runtimeSliceSource[K]
}

func (*SliceGetter[K, V]) reflectTypeParam() reflect.Type {
	return reflect.TypeFor[V]()
}

func (s *SliceGetter[K, V]) setReflectValue(val reflect.Value) error {
	switch v := val.Interface().(type) {
	case []V:
		s.typedValues = v
	case runtimeSliceSource[K]:
		typedValues, ok, err := getRuntimeSliceLiterals[K, V](&v)
		if err != nil {
			return err
		}
		if ok {
			s.typedValues = typedValues
		} else {
			s.runtimeSlice = &v
		}
	default:
		return fmt.Errorf("cannot set value of type %s to a slice of type %s", val.Type(), reflect.TypeFor[V]())
	}
	return nil
}

// getRuntimeSliceLiterals extracts slice literals from a runtimeSliceSource. It returns the
// slice and a boolean indicating if the extraction was successful. The returned values can be
// either scalar or typed getters, which might not hold literal values. In this context, literals
// mean items can be retrieved from the slice without evaluating it. A literal source that does
// not evaluate to a slice returns an error, since evaluating it at runtime would always fail.
func getRuntimeSliceLiterals[K, V any](slice *runtimeSliceSource[K]) ([]V, bool, error) {
	if !slice.isLiteral {
		return nil, false, nil
	}
	sliceValues, err := slice.Get(context.Background(), *new(K))
	if err != nil {
		return nil, false, err
	}
	if sliceValues == nil {
		return nil, true, nil
	}
	if typedValues, ok := sliceValues.([]V); ok {
		return typedValues, true, nil
	}

	var result []V
	if size, ok := slice.sliceLen(sliceValues); ok {
		result = make([]V, 0, size)
	}
	complete := true
	nonNil, err := slice.rangeSlice(sliceValues, func(val any) bool {
		if tv, ok := val.(V); ok {
			result = append(result, tv)
			return true
		}
		complete = false
		return false
	})
	if err != nil {
		return nil, false, err
	}
	if !nonNil {
		return nil, true, nil
	}
	if !complete {
		return nil, false, nil
	}
	return result, true, nil
}

// GetScalarLiteralValues retrieves the literal values from the given slice of scalars.
// If the values cannot be retrieved, it returns the zero value of []V and false.
// A nil slice returns a nil result and true, while an empty slice returns an empty
// non-nil result and true.
// [V] must be a scalar type supported by OTTL slice arguments.
func GetScalarLiteralValues[
	K any,
	V ~uint8 | ~int64 | ~float64 | ~string, // same as buildSliceArg
](slice *SliceGetter[K, V]) ([]V, bool) {
	if slice.runtimeSlice != nil {
		return nil, false
	}
	var result []V
	nonNil, err := slice.Range(
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
	// An empty non-nil slice must not collapse to nil.
	if nonNil && result == nil {
		return []V{}, true
	}
	return result, true
}

// GetLiteralValues retrieves the literal values from the given slice of getters, using
// literalValue to read each item. If an item is not a literal, it returns the zero value
// of []V and false.
// A nil slice returns a nil result and true, while an empty slice returns an empty
// non-nil result and true.
func GetLiteralValues[K, V, G any](slice *SliceGetter[K, G], literalValue func(G) (V, bool)) ([]V, bool) {
	if slice.runtimeSlice != nil {
		return nil, false
	}
	var result []V
	allLiterals := true
	nonNil, err := slice.Range(context.Background(), *new(K), func(value G) bool {
		val, ok := literalValue(value)
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
	// An empty non-nil slice must not collapse to nil.
	if nonNil && result == nil {
		return []V{}, true
	}
	return result, true
}

// Range iterates over elements in the slice and applies the yield function to each item.
// The yield function returns true to continue iterating, false to stop.
//
// The returned boolean reports whether the underlying slice is non-nil, even if
// it is empty or iteration stops early. Ignore the boolean on error.
func (s *SliceGetter[K, V]) Range(ctx context.Context, tCtx K, yield func(value V) bool) (bool, error) {
	if s.runtimeSlice != nil {
		return s.rangeRuntimeSlice(ctx, tCtx, yield)
	}
	return rangeTypedSlice(s.typedValues, yield)
}

// rangeRuntimeSlice iterates over elements in the runtime slice and applies the yield function to each item.
// The returned boolean reports whether the underlying slice is non-nil.
func (s *SliceGetter[K, V]) rangeRuntimeSlice(ctx context.Context, tCtx K, yield func(value V) bool) (bool, error) {
	values, err := s.runtimeSlice.Get(ctx, tCtx)
	if err != nil {
		return false, err
	}
	if values == nil {
		return false, nil
	}

	if typedValues, ok := values.([]V); ok {
		return rangeTypedSlice(typedValues, yield)
	}

	var rangeErr error
	nonNil, err := s.runtimeSlice.rangeSlice(values, func(val any) bool {
		if v, ok := val.(V); ok {
			return yield(v)
		}
		rangeErr = fmt.Errorf("expected slice item of type %s, got %s", reflect.TypeFor[V](), reflect.TypeOf(val))
		return false
	})
	if err != nil {
		return false, err
	}
	if rangeErr != nil {
		return false, rangeErr
	}

	return nonNil, nil
}

// rangeTypedSlice iterates over elements in the typed slice and applies the yield function to each item.
// The returned boolean reports whether the underlying slice is non-nil.
func rangeTypedSlice[V any](typedValues []V, yield func(value V) bool) (bool, error) {
	if typedValues == nil {
		return false, nil
	}
	for _, v := range typedValues {
		if !yield(v) {
			return true, nil
		}
	}
	return true, nil
}

// Len returns the length of the slice when it can be determined without evaluation.
// For literal slices it returns the length and true, otherwise it returns 0 and false.
func (s *SliceGetter[K, V]) Len() (int, bool) {
	if s.typedValues != nil {
		return len(s.typedValues), true
	}
	return 0, false
}

// Get retrieves all values as []V.
// If any slice element is not coercible to V, it returns an error.
func (s *SliceGetter[K, V]) Get(ctx context.Context, tCtx K) ([]V, error) {
	if s.runtimeSlice != nil {
		return s.getRuntimeSliceValue(ctx, tCtx)
	}
	return s.typedValues, nil
}

func (s *SliceGetter[K, V]) getRuntimeSliceValue(ctx context.Context, tCtx K) ([]V, error) {
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
	nonNil, err := s.runtimeSlice.rangeSlice(values, func(val any) bool {
		if v, ok := val.(V); ok {
			result = append(result, v)
			return true
		}
		rangeErr = fmt.Errorf("expected slice item of type %s, got %s", reflect.TypeFor[V](), reflect.TypeOf(val))
		return false
	})
	if err != nil {
		return nil, err
	}
	if rangeErr != nil {
		return nil, rangeErr
	}
	if !nonNil {
		return nil, nil
	}

	return result, nil
}

// sliceElementCoercer is a generic type for handling SliceGetter element coercion operations.
// It stores metadata and provides functionality to coerce and iterate over slice elements.
type sliceElementCoercer[K any] struct {
	sliceItemType        reflect.Type
	sliceItemTypeName    string
	buildSliceItemGetter func(string, Getter[K]) (any, error)
	newLiteral           func(any) Getter[K]
}

func newSliceElementCoercer[K any](
	sliceItemType reflect.Type,
	buildSliceItemGetter func(string, Getter[K]) (any, error),
	newLiteral func(any) Getter[K],
) *sliceElementCoercer[K] {
	return &sliceElementCoercer[K]{
		sliceItemType:        sliceItemType,
		sliceItemTypeName:    sliceItemType.Name(),
		buildSliceItemGetter: buildSliceItemGetter,
		newLiteral:           newLiteral,
	}
}

type runtimeSliceSource[K any] struct {
	Getter[K]
	*sliceElementCoercer[K]
	isLiteral bool
}

// NewRuntimeSliceSource returns a value for SetReflectValue that resolves getter to a slice
// at runtime, or once at parse time when isLiteral is true. newLiteral wraps raw slice items
// as getters, and buildSliceItemGetter converts an item getter to the named element type.
func NewRuntimeSliceSource[K any](
	getter Getter[K],
	isLiteral bool,
	sliceItemType reflect.Type,
	buildSliceItemGetter func(string, Getter[K]) (any, error),
	newLiteral func(any) Getter[K],
) any {
	return runtimeSliceSource[K]{
		Getter:              getter,
		sliceElementCoercer: newSliceElementCoercer(sliceItemType, buildSliceItemGetter, newLiteral),
		isLiteral:           isLiteral,
	}
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
// The returned boolean reports whether the slice is non-nil.
func (c *sliceElementCoercer[K]) rangeSlice(slice any, yield func(val any) bool) (bool, error) {
	switch typedVal := slice.(type) {
	case pcommon.Slice:
		for _, item := range typedVal.All() {
			itemGetter, err := c.buildSliceItemGetter(c.sliceItemTypeName, c.newLiteral(item))
			if err != nil {
				return false, err
			}
			if !yield(itemGetter) {
				return true, nil
			}
		}
	case pcommon.Value:
		if typedVal.Type() != pcommon.ValueTypeSlice {
			return false, fmt.Errorf("expected a slice, got %q", typedVal.Type())
		}
		return c.rangeSlice(typedVal.Slice(), yield)
	default:
		values := reflect.ValueOf(slice)
		if values.Kind() != reflect.Slice {
			return false, fmt.Errorf("expected a slice, got %T", slice)
		}
		if values.IsNil() {
			return false, nil
		}
		for i := 0; i < values.Len(); i++ {
			item := values.Index(i)
			if item.Type() == c.sliceItemType {
				if !yield(item.Interface()) {
					return true, nil
				}
			} else {
				var itemGetter any
				var err error
				rawValue := values.Index(i).Interface()
				if getter, ok := rawValue.(Getter[K]); ok {
					itemGetter, err = c.buildSliceItemGetter(c.sliceItemTypeName, getter)
				} else {
					itemGetter, err = c.buildSliceItemGetter(c.sliceItemTypeName, c.newLiteral(rawValue))
				}
				if err != nil {
					return false, err
				}
				if !yield(itemGetter) {
					return true, nil
				}
			}
		}
	}
	return true, nil
}

// NewTestingSliceGetter creates a SliceGetter that resolves a slice at runtime or uses literals.
func NewTestingSliceGetter[K, T any](literal bool, values []T) *SliceGetter[K, T] {
	createSliceGetter := func(source any) *SliceGetter[K, T] {
		slice := &SliceGetter[K, T]{}
		err := slice.setReflectValue(reflect.ValueOf(source))
		if err != nil {
			panic(err)
		}
		return slice
	}

	if literal {
		return createSliceGetter(values)
	}

	sliceItemType := reflect.TypeFor[T]()
	source := runtimeSliceSource[K]{
		Getter: getterFunc[K](func(context.Context, K) (any, error) {
			return values, nil
		}),
		sliceElementCoercer: newSliceElementCoercer[K](sliceItemType, unsupportedSliceItemGetter[K], nil),
	}
	return createSliceGetter(source)
}

type getterFunc[K any] func(context.Context, K) (any, error)

func (f getterFunc[K]) Get(ctx context.Context, tCtx K) (any, error) {
	return f(ctx, tCtx)
}

// unsupportedSliceItemGetter is never called for testing slices because their values are
// already of the element type.
func unsupportedSliceItemGetter[K any](string, Getter[K]) (any, error) {
	return nil, errors.New("testing slice getters do not coerce slice items")
}
