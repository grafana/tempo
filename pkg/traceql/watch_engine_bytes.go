package traceql

import (
	"fmt"
	math_bits "math/bits"

	"github.com/grafana/tempo/pkg/tempopb"
)

var _ SpanWatcher = (*engineBytesWatcher)(nil)

type engineBytesWatcher struct {
	bytes uint64

	// Runtime fields to avoid allocating closures
	// and escaping to the heap when we call span.AllAttributesFunc.
	attrCallback func(Attribute, Static)
}

// NewEngineBytesWatcher returns a watcher that estimates encoded attribute bytes on matched spans.
// For each watched span it walks AllAttributesFunc and sizes every attribute value (including arrays),
// plus the span start time. The running total is reported under tempopb.AdditionalMetricEngineBytes.
func NewEngineBytesWatcher() SpanWatcher {
	w := &engineBytesWatcher{}
	w.attrCallback = w.watchAttr
	return w
}

func (e *engineBytesWatcher) Conditions() []Condition {
	return nil
}

func (e *engineBytesWatcher) WatchSpan(span Span) bool {
	span.AllAttributesFunc(e.attrCallback)
	if st := span.StartTimeUnixNanos(); st != 0 {
		e.bytes += 1 + uint64(e.varIntSize(st))
	}
	return true // keep watching every span
}

// watchAttr records the size of the attribute.
func (e *engineBytesWatcher) watchAttr(_ Attribute, v Static) {
	// TODO - Include attribute name?
	e.bytes += uint64(e.staticSize(v))
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
	return map[string]int64{tempopb.AdditionalMetricEngineBytes: int64(e.bytes)}
}
