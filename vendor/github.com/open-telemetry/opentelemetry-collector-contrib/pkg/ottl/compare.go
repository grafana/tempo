// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"bytes"

	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
)

// The functions in this file implement a general-purpose comparison of two
// values of type any, which for the purposes of OTTL mean values that are one of
// int, float, string, bool, or pointers to those, or []byte, or nil.

// invalidComparison returns false for everything except NE (where it returns true to indicate that the
// objects were definitely not equivalent).
// It also gives us an opportunity to log something.
func (p *Parser[K]) invalidComparison(msg string, op compareOp) bool {
	p.telemetrySettings.Logger.Debug(msg, zap.Any("op", op))
	return op == NE
}

// comparePrimitives implements a generic comparison helper for all Ordered types (derived from Float, Int, or string).
// According to benchmarks, it's faster than explicit comparison functions for these types.
func comparePrimitives[T constraints.Ordered](a T, b T, op compareOp) bool {
	switch op {
	case EQ:
		return a == b
	case NE:
		return a != b
	case LT:
		return a < b
	case LTE:
		return a <= b
	case GTE:
		return a >= b
	case GT:
		return a > b
	default:
		return false
	}
}

func compareBools(a bool, b bool, op compareOp) bool {
	switch op {
	case EQ:
		return a == b
	case NE:
		return a != b
	case LT:
		return !a && b
	case LTE:
		return !a || b
	case GTE:
		return a || !b
	case GT:
		return a && !b
	default:
		return false
	}
}

func compareBytes(a []byte, b []byte, op compareOp) bool {
	switch op {
	case EQ:
		return bytes.Equal(a, b)
	case NE:
		return !bytes.Equal(a, b)
	case LT:
		return bytes.Compare(a, b) < 0
	case LTE:
		return bytes.Compare(a, b) <= 0
	case GTE:
		return bytes.Compare(a, b) >= 0
	case GT:
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
		return p.invalidComparison("bool to non-bool", op)
	}
}

func (p *Parser[K]) compareString(a string, b any, op compareOp) bool {
	switch v := b.(type) {
	case string:
		return comparePrimitives(a, v, op)
	default:
		return p.invalidComparison("string to non-string", op)
	}
}

func (p *Parser[K]) compareByte(a []byte, b any, op compareOp) bool {
	switch v := b.(type) {
	case nil:
		return op == NE
	case []byte:
		if v == nil {
			return op == NE
		}
		return compareBytes(a, v, op)
	default:
		return p.invalidComparison("Bytes to non-Bytes", op)
	}
}

func (p *Parser[K]) compareInt64(a int64, b any, op compareOp) bool {
	switch v := b.(type) {
	case int64:
		return comparePrimitives(a, v, op)
	case float64:
		return comparePrimitives(float64(a), v, op)
	default:
		return p.invalidComparison("int to non-numeric value", op)
	}
}

func (p *Parser[K]) compareFloat64(a float64, b any, op compareOp) bool {
	switch v := b.(type) {
	case int64:
		return comparePrimitives(a, float64(v), op)
	case float64:
		return comparePrimitives(a, v, op)
	default:
		return p.invalidComparison("float to non-numeric value", op)
	}
}

// a and b are the return values from a Getter; we try to compare them
// according to the given operator.
func (p *Parser[K]) compare(a any, b any, op compareOp) bool {
	// nils are equal to each other and never equal to anything else,
	// so if they're both nil, report equality.
	if a == nil && b == nil {
		return op == EQ || op == LTE || op == GTE
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
	default:
		// If we don't know what type it is, we can't do inequalities yet. So we can fall back to the old behavior where we just
		// use Go's standard equality.
		switch op {
		case EQ:
			return a == b
		case NE:
			return a != b
		default:
			return p.invalidComparison("unsupported type for inequality on left", op)
		}
	}
}
