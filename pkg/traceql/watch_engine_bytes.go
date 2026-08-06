package traceql

import (
	"fmt"
	math_bits "math/bits"
	"sync/atomic"

	"github.com/grafana/tempo/pkg/tempopb"
)

var _ SpanWatcher = (*engineBytesWatcher)(nil)

type engineBytesWatcher struct {
	bytes atomic.Int64
}

// NewEngineBytesWatcher returns a watcher that estimates encoded attribute bytes on matched spans.
// For each watched span it walks AllAttributesFunc and sizes every attribute value (including arrays),
// plus the span start time. The running total is reported under tempopb.AdditionalMetricEngineBytes.
func NewEngineBytesWatcher() SpanWatcher {
	return &engineBytesWatcher{}
}

func (e *engineBytesWatcher) Conditions() []Condition {
	return nil
}

func (e *engineBytesWatcher) WatchSpan(span Span) bool {
	span.AllAttributesFunc(func(_ Attribute, v Static) {
		e.bytes.Add(int64(e.staticSize(v)))
	})
	if st := span.StartTimeUnixNanos(); st != 0 {
		e.bytes.Add(1 + int64(e.varIntSize(st)))
	}
	return true // keep watching every span
}

// staticSize returns the encoded size of a Static value.
// Scalars: 1 type byte + payload.
// Arrays: 1 type byte + length varint + element payloads (no per-element type byte).
func (e *engineBytesWatcher) staticSize(v Static) int {
	switch v.Type {
	case TypeNil:
		return 1
	case TypeString:
		l := len(v.valBytes)
		return 1 + l + e.varIntSize(uint64(l))
	case TypeInt, TypeStatus, TypeKind, TypeDuration:
		return 1 + e.varIntSize(v.valScalar)
	case TypeFloat:
		return 1 + 8
	case TypeBoolean:
		return 1 + 1
	case TypeIntArray:
		ints, _ := v.IntArray()
		n := 1 + e.varIntSize(uint64(len(ints)))
		for _, i := range ints {
			n += e.varIntSize(uint64(i))
		}
		return n
	case TypeFloatArray:
		floats, _ := v.FloatArray()
		return 1 + e.varIntSize(uint64(len(floats))) + 8*len(floats)
	case TypeStringArray:
		strs, _ := v.StringArray()
		n := 1 + e.varIntSize(uint64(len(strs)))
		for _, s := range strs {
			l := len(s)
			n += l + e.varIntSize(uint64(l))
		}
		return n
	case TypeBooleanArray:
		bools, _ := v.BooleanArray()
		return 1 + e.varIntSize(uint64(len(bools))) + len(bools)
	default:
		panic(fmt.Sprintf("Unhandled attribute type so far: %v", v.Type))
	}
}

func (*engineBytesWatcher) varIntSize(v uint64) int {
	return (math_bits.Len64(v|1) + 6) / 7
}

func (e *engineBytesWatcher) Active() bool {
	return true
}

func (e *engineBytesWatcher) Stats() map[string]int64 {
	return map[string]int64{tempopb.AdditionalMetricEngineBytes: e.bytes.Load()}
}
