// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"golang.org/x/exp/constraints"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	typeNotIndexableError = "type %T does not support indexing"
)

var (
	errMissingSetKey = errors.New("cannot set slice value without key")
	errMissingGetKey = errors.New("cannot get slice value without key")
)

func getSliceIndexFromKeys[K any](ctx context.Context, tCtx K, sliceLen int, keys []ottl.Key[K]) (int, error) {
	i, err := keys[0].Int(ctx, tCtx)
	if err != nil {
		return 0, err
	}
	if i == nil {
		resInt, err := FetchValueFromExpression[K, int64](ctx, tCtx, keys[0])
		if err != nil {
			return 0, fmt.Errorf("unable to resolve an integer index in slice: %w", err)
		}
		i = resInt
	}

	idx := int(*i)

	if idx < 0 || idx >= sliceLen {
		return 0, fmt.Errorf("index %d out of bounds", idx)
	}

	return idx, nil
}

func GetSliceValue[K any](ctx context.Context, tCtx K, s pcommon.Slice, keys []ottl.Key[K]) (any, error) {
	if len(keys) == 0 {
		return 0, errMissingGetKey
	}

	idx, err := getSliceIndexFromKeys(ctx, tCtx, s.Len(), keys)
	if err != nil {
		return nil, err
	}

	return getIndexableValue[K](ctx, tCtx, s.At(idx), keys[1:])
}

func SetSliceValue[K any](ctx context.Context, tCtx K, s pcommon.Slice, keys []ottl.Key[K], val any) error {
	if len(keys) == 0 {
		return errMissingSetKey
	}

	idx, err := getSliceIndexFromKeys(ctx, tCtx, s.Len(), keys)
	if err != nil {
		return err
	}

	return SetIndexableValue[K](ctx, tCtx, s.At(idx), val, keys[1:])
}

// CommonTypedSlice is an interface for typed pdata slices, such as pcommon.StringSlice,
// pcommon.Int64Slice, pcommon.Int32Slice, etc.
type CommonTypedSlice[T any] interface {
	At(int) T
	Len() int
	FromRaw(val []T)
	SetAt(int, T)
	AsRaw() []T
}

// GetCommonTypedSliceValue is like GetSliceValue, but for retrieving a value from a pdata
// typed slice. [V] is the type of the slice elements. If no keys are provided, it returns
// an error. This function does not support slice elements indexing.
func GetCommonTypedSliceValue[K, V any](ctx context.Context, tCtx K, s CommonTypedSlice[V], keys []ottl.Key[K]) (V, error) {
	if len(keys) == 0 {
		return *new(V), errMissingGetKey
	}
	if len(keys) > 1 {
		return *new(V), fmt.Errorf(typeNotIndexableError, s)
	}

	idx, err := getSliceIndexFromKeys(ctx, tCtx, s.Len(), keys)
	if err != nil {
		return *new(V), err
	}

	return s.At(idx), nil
}

// SetCommonTypedSliceValue sets the value of a pdata typed slice element. [V] is the type
// of the slice elements. The any-wrapped value type must be [V], otherwise an error is
// returned. This function does not support slice elements indexing.
func SetCommonTypedSliceValue[K, V any](ctx context.Context, tCtx K, s CommonTypedSlice[V], keys []ottl.Key[K], val any) error {
	if len(keys) == 0 {
		return errMissingSetKey
	} else if len(keys) > 1 {
		return fmt.Errorf(typeNotIndexableError, s)
	}

	idx, err := getSliceIndexFromKeys(ctx, tCtx, s.Len(), keys)
	if err != nil {
		return err
	}

	typeVal, ok := val.(V)
	if !ok {
		return fmt.Errorf("invalid value type provided for a slice of %T: %T", *new(V), val)
	}

	s.SetAt(idx, typeVal)
	return nil
}

// SetCommonTypedSliceValues sets the value of a pdata typed slice. It does handle all
// different input types OTTL generate, such as []V, []any, or a pcommon.Slice.
// If the value is a slice of [any] or pcommon.Slice, and it has an element that the type
// is not [V], an error is returned.
func SetCommonTypedSliceValues[V any](s CommonTypedSlice[V], val any) error {
	switch typeVal := val.(type) {
	case CommonTypedSlice[V]:
		s.FromRaw(typeVal.AsRaw())
	case []V:
		s.FromRaw(typeVal)
	case []any:
		raw := make([]V, len(typeVal))
		for i, v := range typeVal {
			sv, ok := v.(V)
			if !ok {
				return fmt.Errorf("invalid value type provided for a slice of %T: %T", *new(V), v)
			}
			raw[i] = sv
		}
		s.FromRaw(raw)
	case pcommon.Slice:
		raw := make([]V, typeVal.Len())
		for i := range typeVal.Len() {
			v, ok := typeVal.At(i).AsRaw().(V)
			if !ok {
				return fmt.Errorf("invalid value type provided for a slice of %T: %T", raw, typeVal.At(i).AsRaw())
			}
			raw[i] = v
		}
		s.FromRaw(raw)
	default:
		return fmt.Errorf("invalid type provided for setting a slice of %T: %T", val, *new(V))
	}

	return nil
}

// GetCommonIntSliceValues converts a pdata typed slice of [constraints.Integer] into
// []int64, which is the standard OTTL type for integer slices.
func GetCommonIntSliceValues[V constraints.Integer](val CommonTypedSlice[V]) []int64 {
	output := make([]int64, val.Len())
	for i := range val.Len() {
		output[i] = int64(val.At(i))
	}
	return output
}

// GetCommonIntSliceValue is like GetCommonTypedSliceValue, but for integer pdata typed
// slices.
func GetCommonIntSliceValue[K any, V constraints.Integer](ctx context.Context, tCtx K, s CommonTypedSlice[V], keys []ottl.Key[K]) (int64, error) {
	value, err := GetCommonTypedSliceValue[K, V](ctx, tCtx, s, keys)
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

// SetCommonIntSliceValue is like SetCommonTypedSliceValue, but for integer pdata typed
// slice element values. [V] is the type of the slice elements.
// The any-wrapped value type must be and integer convertible to [V], otherwise an error
// is returned. This function does not support slice elements indexing.
func SetCommonIntSliceValue[K any, V constraints.Integer](ctx context.Context, tCtx K, s CommonTypedSlice[V], keys []ottl.Key[K], val any) error {
	var intVal V
	switch typeVal := val.(type) {
	case int:
		intVal = V(typeVal)
	case int8:
		intVal = V(typeVal)
	case int16:
		intVal = V(typeVal)
	case int32:
		intVal = V(typeVal)
	case int64:
		intVal = V(typeVal)
	case uint:
		intVal = V(typeVal)
	case uint8:
		intVal = V(typeVal)
	case uint16:
		intVal = V(typeVal)
	case uint32:
		intVal = V(typeVal)
	case uint64:
		intVal = V(typeVal)
	default:
		return fmt.Errorf("invalid type provided for setting a slice of %T: %T", *new(V), val)
	}

	return SetCommonTypedSliceValue[K, V](ctx, tCtx, s, keys, intVal)
}

// SetCommonIntSliceValues is like SetCommonTypedSliceValues, but for integer pdata typed
// slices. [V] is the type of the slice elements. The value must be []any, []int64, []T,
// or a pcommon.Slice which elements are type inferable to int64, otherwise an error is
// returned.
func SetCommonIntSliceValues[V constraints.Integer](s CommonTypedSlice[V], val any) error {
	switch typeVal := val.(type) {
	case CommonTypedSlice[V]:
		s.FromRaw(typeVal.AsRaw())
	case []int64:
		raw := make([]V, len(typeVal))
		for i, v := range typeVal {
			raw[i] = V(v)
		}
		s.FromRaw(raw)
	case []V:
		s.FromRaw(typeVal)
	case []any:
		raw := make([]V, len(typeVal))
		for i, v := range typeVal {
			iv, ok := v.(int64)
			if !ok {
				return fmt.Errorf("invalid value type provided for a slice of %T: %T, expected int64", *new(V), v)
			}
			raw[i] = V(iv)
		}
		s.FromRaw(raw)
	case pcommon.Slice:
		raw := make([]V, typeVal.Len())
		for i := range typeVal.Len() {
			v, ok := typeVal.At(i).AsRaw().(int64)
			if !ok {
				return fmt.Errorf("invalid value type provided for slice of %T: %T, expected int64", *new(V), typeVal.At(i).AsRaw())
			}
			raw[i] = V(v)
		}
		s.FromRaw(raw)
	default:
		return fmt.Errorf("cannot set a slice of %T from a value type: %T", val, *new(V))
	}

	return nil
}
