// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"bytes"
	"reflect"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"golang.org/x/exp/constraints"
)

// ValueComparator defines methods for comparing values using the OTTL comparison rules
// (https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/LANGUAGE.md#comparison-rules)
type ValueComparator interface {
	// Equal compares two values for equality, returning true if they are equals
	// according to the OTTL comparison rules.
	Equal(a any, b any) bool
	// NotEqual compares two values for equality, returning true if they are different
	// according to the OTTL comparison rules.
	NotEqual(a any, b any) bool
	// Less compares two values, returning true if the first value is less than the second
	// value, using the OTTL comparison rules.
	Less(a any, b any) bool
	// LessEqual compares two values, returning true if the first value is less or equal
	// to the second value, using the OTTL comparison rules.
	LessEqual(a any, b any) bool
	// Greater compares two values, returning true if the first value is greater than the
	// second value, using the OTTL comparison rules.
	Greater(a any, b any) bool
	// GreaterEqual compares two values, returning true if the first value is greater or
	// equal to the second value, using the OTTL comparison rules.
	GreaterEqual(a any, b any) bool
	// compare is a private method that compares two values using the grammar compareOp,
	// it also restricts custom implementations outside of this package.
	compare(a any, b any, op compareOp) bool
}

// ottlValueComparator is the default implementation of the ValueComparator
type ottlValueComparator struct{}

// The functions in this file implement a general-purpose comparison of two
// values of type any, which for the purposes of OTTL mean values that are one of
// int, float, string, bool, or pointers to those, or []byte, or nil.

// invalidComparison returns false for everything except ne (where it returns true to indicate that the
// objects were definitely not equivalent).
func (p *ottlValueComparator) invalidComparison(op compareOp) bool {
	return op == ne
}

// comparePrimitives implements a generic comparison helper for all Ordered types (derived from Float, Int, or string).
// According to benchmarks, it's faster than explicit comparison functions for these types.
func comparePrimitives[T constraints.Ordered](a T, b T, op compareOp) bool {
	switch op {
	case eq:
		return a == b
	case ne:
		return a != b
	case lt:
		return a < b
	case lte:
		return a <= b
	case gte:
		return a >= b
	case gt:
		return a > b
	default:
		return false
	}
}

func (p *ottlValueComparator) compareBools(a bool, b bool, op compareOp) bool {
	switch op {
	case eq:
		return a == b
	case ne:
		return a != b
	case lt:
		return !a && b
	case lte:
		return !a || b
	case gte:
		return a || !b
	case gt:
		return a && !b
	default:
		return false
	}
}

func (p *ottlValueComparator) compareBytes(a []byte, b []byte, op compareOp) bool {
	switch op {
	case eq:
		return bytes.Equal(a, b)
	case ne:
		return !bytes.Equal(a, b)
	case lt:
		return bytes.Compare(a, b) < 0
	case lte:
		return bytes.Compare(a, b) <= 0
	case gte:
		return bytes.Compare(a, b) >= 0
	case gt:
		return bytes.Compare(a, b) > 0
	default:
		return false
	}
}

func (p *ottlValueComparator) compareBool(a bool, b any, op compareOp) bool {
	switch v := b.(type) {
	case bool:
		return p.compareBools(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareString(a string, b any, op compareOp) bool {
	switch v := b.(type) {
	case string:
		return comparePrimitives(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareByte(a []byte, b any, op compareOp) bool {
	switch v := b.(type) {
	case nil:
		return op == ne
	case []byte:
		if v == nil {
			return op == ne
		}
		return p.compareBytes(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareInt64(a int64, b any, op compareOp) bool {
	switch v := b.(type) {
	case int64:
		return comparePrimitives(a, v, op)
	case float64:
		return comparePrimitives(float64(a), v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareFloat64(a float64, b any, op compareOp) bool {
	switch v := b.(type) {
	case int64:
		return comparePrimitives(a, float64(v), op)
	case float64:
		return comparePrimitives(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareDuration(a time.Duration, b any, op compareOp) bool {
	switch v := b.(type) {
	case time.Duration:
		ansecs := a.Nanoseconds()
		vnsecs := v.Nanoseconds()
		return comparePrimitives(ansecs, vnsecs, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareTime(a time.Time, b any, op compareOp) bool {
	switch v := b.(type) {
	case time.Time:
		switch op {
		case eq:
			return a.Equal(v)
		case ne:
			return !a.Equal(v)
		case lt:
			return a.Before(v)
		case lte:
			return a.Before(v) || a.Equal(v)
		case gte:
			return a.After(v) || a.Equal(v)
		case gt:
			return a.After(v)
		default:
			return p.invalidComparison(op)
		}
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareMap(a map[string]any, b any, op compareOp) bool {
	switch v := b.(type) {
	case pcommon.Map:
		switch op {
		case eq:
			return reflect.DeepEqual(a, v.AsRaw())
		case ne:
			return !reflect.DeepEqual(a, v.AsRaw())
		default:
			return p.invalidComparison(op)
		}
	case map[string]any:
		switch op {
		case eq:
			return reflect.DeepEqual(a, v)
		case ne:
			return !reflect.DeepEqual(a, v)
		default:
			return p.invalidComparison(op)
		}
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) comparePMap(a pcommon.Map, b any, op compareOp) bool {
	switch v := b.(type) {
	case pcommon.Map:
		switch op {
		case eq:
			return a.Equal(v)
		case ne:
			return !a.Equal(v)
		default:
			return p.invalidComparison(op)
		}
	case map[string]any:
		return p.compareMap(a.AsRaw(), v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) compareSlice(a []any, b any, op compareOp) bool {
	switch v := b.(type) {
	case pcommon.Slice:
		switch op {
		case eq:
			return reflect.DeepEqual(a, v.AsRaw())
		case ne:
			return !reflect.DeepEqual(a, v.AsRaw())
		default:
			return p.invalidComparison(op)
		}
	case []any:
		switch op {
		case eq:
			return reflect.DeepEqual(a, v)
		case ne:
			return !reflect.DeepEqual(a, v)
		default:
			return p.invalidComparison(op)
		}
	default:
		return p.invalidComparison(op)
	}
}

func (p *ottlValueComparator) comparePSlice(a pcommon.Slice, b any, op compareOp) bool {
	switch v := b.(type) {
	case pcommon.Slice:
		switch op {
		case eq:
			return a.Equal(v)
		case ne:
			return !a.Equal(v)
		default:
			return p.invalidComparison(op)
		}
	case []any:
		return p.compareSlice(a.AsRaw(), v, op)
	default:
		return p.invalidComparison(op)
	}
}

// a and b are the return values from a Getter; we try to compare them
// according to the given operator.
func (p *ottlValueComparator) compare(a any, b any, op compareOp) bool {
	// nils are equal to each other and never equal to anything else,
	// so if they're both nil, report equality.
	if a == nil && b == nil {
		return op == eq || op == lte || op == gte
	}
	// Anything else, we switch on the left side first.
	switch v := a.(type) {
	case nil:
		// If a was nil, it means b wasn't and inequalities don't apply,
		// so let's swap and give it the chance to get evaluated.
		return p.compare(b, nil, op)
	case bool:
		return p.compareBool(v, b, op)
	case int64:
		return p.compareInt64(v, b, op)
	case float64:
		return p.compareFloat64(v, b, op)
	case string:
		return p.compareString(v, b, op)
	case []byte:
		if v == nil {
			return p.compare(b, nil, op)
		}
		return p.compareByte(v, b, op)
	case time.Duration:
		return p.compareDuration(v, b, op)
	case time.Time:
		return p.compareTime(v, b, op)
	case map[string]any:
		return p.compareMap(v, b, op)
	case pcommon.Map:
		return p.comparePMap(v, b, op)
	case []any:
		return p.compareSlice(v, b, op)
	case pcommon.Slice:
		return p.comparePSlice(v, b, op)
	default:
		// If we don't know what type it is, we can't do inequalities yet. So we can fall back to the old behavior where we just
		// use Go's standard equality.
		switch op {
		case eq:
			return a == b
		case ne:
			return a != b
		default:
			return p.invalidComparison(op)
		}
	}
}

func (p *ottlValueComparator) Equal(a any, b any) bool {
	return p.compare(a, b, eq)
}

func (p *ottlValueComparator) NotEqual(a any, b any) bool {
	return p.compare(a, b, ne)
}

func (p *ottlValueComparator) Less(a any, b any) bool {
	return p.compare(a, b, lt)
}

func (p *ottlValueComparator) LessEqual(a any, b any) bool {
	return p.compare(a, b, lte)
}

func (p *ottlValueComparator) Greater(a any, b any) bool {
	return p.compare(a, b, gt)
}

func (p *ottlValueComparator) GreaterEqual(a any, b any) bool {
	return p.compare(a, b, gte)
}

// NewValueComparator creates a new ValueComparator instance that can be used to compare
// values using the OTTL comparison rules.
func NewValueComparator() ValueComparator {
	return &ottlValueComparator{}
}
