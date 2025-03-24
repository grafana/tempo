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

// The functions in this file implement a general-purpose comparison of two
// values of type any, which for the purposes of OTTL mean values that are one of
// int, float, string, bool, or pointers to those, or []byte, or nil.

// invalidComparison returns false for everything except ne (where it returns true to indicate that the
// objects were definitely not equivalent).
func (p *Parser[K]) invalidComparison(op compareOp) bool {
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

func compareBools(a bool, b bool, op compareOp) bool {
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

func compareBytes(a []byte, b []byte, op compareOp) bool {
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

func (p *Parser[K]) compareBool(a bool, b any, op compareOp) bool {
	switch v := b.(type) {
	case bool:
		return compareBools(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *Parser[K]) compareString(a string, b any, op compareOp) bool {
	switch v := b.(type) {
	case string:
		return comparePrimitives(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *Parser[K]) compareByte(a []byte, b any, op compareOp) bool {
	switch v := b.(type) {
	case nil:
		return op == ne
	case []byte:
		if v == nil {
			return op == ne
		}
		return compareBytes(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *Parser[K]) compareInt64(a int64, b any, op compareOp) bool {
	switch v := b.(type) {
	case int64:
		return comparePrimitives(a, v, op)
	case float64:
		return comparePrimitives(float64(a), v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *Parser[K]) compareFloat64(a float64, b any, op compareOp) bool {
	switch v := b.(type) {
	case int64:
		return comparePrimitives(a, float64(v), op)
	case float64:
		return comparePrimitives(a, v, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *Parser[K]) compareDuration(a time.Duration, b any, op compareOp) bool {
	switch v := b.(type) {
	case time.Duration:
		ansecs := a.Nanoseconds()
		vnsecs := v.Nanoseconds()
		return comparePrimitives(ansecs, vnsecs, op)
	default:
		return p.invalidComparison(op)
	}
}

func (p *Parser[K]) compareTime(a time.Time, b any, op compareOp) bool {
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

func (p *Parser[K]) compareMap(a map[string]any, b any, op compareOp) bool {
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

func (p *Parser[K]) comparePMap(a pcommon.Map, b any, op compareOp) bool {
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

// a and b are the return values from a Getter; we try to compare them
// according to the given operator.
func (p *Parser[K]) compare(a any, b any, op compareOp) bool {
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
