// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/slicegetter"
)

// SliceGetter represents a slice argument in OTTL functions. Unlike bare []V
// parameters, which only accept literal lists [value], it also accepts a single
// getter (path or expression) that resolves to []V at runtime.
// V is the element type of the resolved slice. It may be a typed Getter
// (e.g.: ottl.StringGetter[K]) or a scalar type supported by OTTL.
//
// Stable OTTL functions only accept paths and expressions for a SliceGetter
// argument when the pkg.ottl.functions.enableDynamicSliceArguments feature gate is enabled.
type SliceGetter[K, V any] = slicegetter.SliceGetter[K, V]

// GetScalarLiteralValues retrieves the literal values from the given slice of scalars.
// If the values cannot be retrieved, it returns the zero value of []V and false.
// [V] must be a scalar type supported by OTTL slice arguments.
func GetScalarLiteralValues[
	K any,
	V ~uint8 | ~int64 | ~float64 | ~string,
](slice *SliceGetter[K, V]) ([]V, bool) {
	return slicegetter.GetScalarLiteralValues(slice)
}

// GetLiteralValues retrieves the literal values from the given slice of getters.
// If the getter or the value it's currently holding is not a literal value, it
// returns the zero value of []V and false.
// [G] is the type of the slice elements, which must be typed Getter.
// [V] is the expected type of the slice values.
func GetLiteralValues[K, V any, G ottl.TypedGetter[K, V]](slice *SliceGetter[K, G]) ([]V, bool) {
	return slicegetter.GetLiteralValues(slice, func(getter G) (V, bool) {
		return ottl.GetLiteralValue[K, V](getter)
	})
}

// NewTestingSliceGetter creates a SliceGetter for use in tests. When literal is true the
// values are stored as a literal list, otherwise they are resolved at runtime.
func NewTestingSliceGetter[K, T any](literal bool, values []T) *SliceGetter[K, T] {
	return slicegetter.NewTestingSliceGetter[K](literal, values)
}
