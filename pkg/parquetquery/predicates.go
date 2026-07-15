package parquetquery

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/grafana/tempo/pkg/regexp"

	pq "github.com/parquet-go/parquet-go"
)

// Predicate is a pushdown predicate evaluated at the value level (KeepValue) and,
// for chunk/page skipping, at the value-range level (KeepRange). Chunk-, page-,
// dictionary-, and null-level skipping are handled generically by the iterator
// layer (keepColumnChunk / keepPage) using these two methods.
type Predicate interface {
	fmt.Stringer

	KeepValue(pq.Value) bool
	// KeepRange reports whether any value in the inclusive [min,max] range could match.
	KeepRange(min, max pq.Value) bool
}

// NewStringEqualPredicate is just an alias for the equivalent byte predicate
func NewStringEqualPredicate(s string) Predicate {
	return NewByteEqualPredicate([]byte(s))
}

// NewStringNotEqualPredicate is just an alias for the equivalent byte predicate
func NewStringNotEqualPredicate(s string) Predicate {
	return NewByteNotEqualPredicate([]byte(s))
}

// byteInMapThreshold is the number of values at or above which KeepValue uses a
// map lookup instead of a linear scan. Below it the scan is faster (lower overhead).
const byteInMapThreshold = 8

// ByteInPredicate checks for any of the given strings. Case-sensitive exact byte matching
type ByteInPredicate struct {
	values    [][]byte
	valuesMap map[string]struct{} // set when len(values) >= byteInMapThreshold
}

var _ Predicate = (*ByteInPredicate)(nil)

func NewStringInPredicate(ss []string) Predicate {
	bb := make([][]byte, len(ss))
	for i := range ss {
		bb[i] = []byte(ss[i])
	}
	return NewByteInPredicate(bb)
}

func NewByteInPredicate(bb [][]byte) Predicate {
	p := &ByteInPredicate{values: bb}
	if len(bb) >= byteInMapThreshold {
		p.valuesMap = make(map[string]struct{}, len(bb))
		for _, b := range bb {
			p.valuesMap[string(b)] = struct{}{}
		}
	}
	return p
}

func (p *ByteInPredicate) String() string {
	var strs string
	for i, s := range p.values {
		if i > 0 {
			strs += ", "
		}
		strs += string(s)
	}
	return fmt.Sprintf("ByteInPredicate{%s}", strs)
}

func (p *ByteInPredicate) KeepValue(v pq.Value) bool {
	ba := v.ByteArray()
	if p.valuesMap != nil {
		_, ok := p.valuesMap[string(ba)]
		return ok
	}
	for _, ss := range p.values {
		if bytes.Equal(ba, ss) {
			return true
		}
	}
	return false
}

func (p *ByteInPredicate) KeepRange(min, max pq.Value) bool {
	lo, hi := min.ByteArray(), max.ByteArray()
	for _, s := range p.values {
		if bytes.Compare(lo, s) <= 0 && bytes.Compare(hi, s) >= 0 {
			return true
		}
	}
	return false
}

// ByteNotInPredicate checks for any of the given strings. Case-sensitive exact byte matching
type ByteNotInPredicate struct {
	values    [][]byte
	valuesMap map[string]struct{} // set when len(values) >= byteInMapThreshold
}

var _ Predicate = (*ByteNotInPredicate)(nil)

func NewStringNotInPredicate(ss []string) Predicate {
	bb := make([][]byte, len(ss))
	for i := range ss {
		bb[i] = []byte(ss[i])
	}
	return NewByteNotInPredicate(bb)
}

func NewByteNotInPredicate(bb [][]byte) Predicate {
	p := &ByteNotInPredicate{values: bb}
	if len(bb) >= byteInMapThreshold {
		p.valuesMap = make(map[string]struct{}, len(bb))
		for _, b := range bb {
			p.valuesMap[string(b)] = struct{}{}
		}
	}
	return p
}

func (p *ByteNotInPredicate) String() string {
	var strs string
	for i, s := range p.values {
		if i > 0 {
			strs += ", "
		}
		strs += string(s)
	}
	return fmt.Sprintf("ByteNotInPredicate{%s}", strs)
}

func (p *ByteNotInPredicate) KeepValue(v pq.Value) bool {
	ba := v.ByteArray()
	if p.valuesMap != nil {
		_, ok := p.valuesMap[string(ba)]
		return !ok
	}
	for _, ss := range p.values {
		if bytes.Equal(ba, ss) {
			return false
		}
	}
	return true
}

func (p *ByteNotInPredicate) KeepRange(pq.Value, pq.Value) bool { return true }

type regexPredicate struct {
	matcher *regexp.Regexp
}

var _ Predicate = (*regexPredicate)(nil)

// NewRegexInPredicate checks for match against any of the given regexs.
// Memoized and resets on each row group.
func NewRegexInPredicate(regs []string) (Predicate, error) {
	return newRegexPredicate(regs, true)
}

// NewRegexNotInPredicate checks for values that not match against any of the given regexs.
// Memoized and resets on each row group.
func NewRegexNotInPredicate(regs []string) (Predicate, error) {
	return newRegexPredicate(regs, false)
}

func newRegexPredicate(regs []string, shouldMatch bool) (Predicate, error) {
	m, err := regexp.NewRegexp(regs, shouldMatch)
	if err != nil {
		return nil, err
	}

	return &regexPredicate{
		matcher: m,
	}, nil
}

func (p *regexPredicate) String() string {
	return fmt.Sprintf("RegexPredicate{%s}", p.matcher.String())
}

func (p *regexPredicate) keep(v *pq.Value) bool {
	if v.IsNull() {
		return false
	}

	return p.matcher.Match(v.ByteArray())
}

func (p *regexPredicate) KeepValue(v pq.Value) bool {
	return p.keep(&v)
}

func (p *regexPredicate) KeepRange(pq.Value, pq.Value) bool { return true }

type SubstringPredicate struct {
	substring []byte
	matches   map[string]bool
}

var _ Predicate = (*SubstringPredicate)(nil)

func NewSubstringPredicate(substring string) *SubstringPredicate {
	return &SubstringPredicate{
		substring: []byte(substring),
		matches:   map[string]bool{},
	}
}

func (p *SubstringPredicate) String() string {
	return fmt.Sprintf("SubstringPredicate{%s}", p.substring)
}

func (p *SubstringPredicate) KeepValue(v pq.Value) bool {
	b := v.ByteArray()

	// Check uses zero alloc optimization of map[string([]byte)]
	if matched, ok := p.matches[string(b)]; ok {
		return matched
	}

	matched := bytes.Contains(b, p.substring)

	// Only alloc the string when updating the map
	p.matches[string(b)] = matched

	return matched
}

func (p *SubstringPredicate) KeepRange(pq.Value, pq.Value) bool { return true }

// IntBetweenPredicate checks for int between the bounds [min,max] inclusive
type IntBetweenPredicate struct {
	min, max int64
}

var _ Predicate = (*IntBetweenPredicate)(nil)

func NewIntBetweenPredicate(min, max int64) *IntBetweenPredicate {
	return &IntBetweenPredicate{min, max}
}

func (p *IntBetweenPredicate) String() string {
	return fmt.Sprintf("IntBetweenPredicate{%d,%d}", p.min, p.max)
}

func (p *IntBetweenPredicate) KeepValue(v pq.Value) bool {
	vv := v.Int64()
	return p.min <= vv && vv <= p.max
}

func (p *IntBetweenPredicate) KeepRange(min, max pq.Value) bool {
	return p.max >= min.Int64() && p.min <= max.Int64()
}

// GenericPredicate with callbacks to evaluate data of type T
// Fn evaluates a single data point and is required. Optionally,
// a RangeFn can evaluate a min/max range and is used to
// skip column chunks and pages when RangeFn is supplied and
// the column chunk or page also include bounds metadata.
type GenericPredicate[T any] struct {
	Fn      func(T) bool
	RangeFn func(min, max T) bool
	Extract func(pq.Value) T
}

var _ Predicate = (*GenericPredicate[int64])(nil)

// NewGenericPredicate is deprecated due to speed concerns. Please use a predicated hard coded to the type you are working with.
// If no such predicate exists add it to the generator in ../parquetquerygen/predicates.go
func NewGenericPredicate[T any](fn func(T) bool, rangeFn func(T, T) bool, extract func(pq.Value) T) *GenericPredicate[T] {
	return &GenericPredicate[T]{Fn: fn, RangeFn: rangeFn, Extract: extract}
}

func (p *GenericPredicate[T]) String() string {
	return "GenericPredicate{}"
}

func (p *GenericPredicate[T]) KeepValue(v pq.Value) bool {
	return p.Fn(p.Extract(v))
}

func (p *GenericPredicate[T]) KeepRange(min, max pq.Value) bool {
	if p.RangeFn == nil {
		return true
	}
	return p.RangeFn(p.Extract(min), p.Extract(max))
}

type OrPredicate struct {
	preds []Predicate
}

var _ Predicate = (*OrPredicate)(nil)

func NewOrPredicate(preds ...Predicate) *OrPredicate {
	return &OrPredicate{
		preds: preds,
	}
}

func (p *OrPredicate) String() string {
	var preds []string
	for _, pred := range p.preds {
		if pred != nil {
			preds = append(preds, pred.String())
		} else {
			preds = append(preds, "nil")
		}
	}
	return fmt.Sprintf("OrPredicate{%s}", strings.Join(preds, ","))
}

func (p *OrPredicate) KeepValue(v pq.Value) bool {
	for _, p := range p.preds {
		if p == nil {
			// Nil means all values are returned
			return true
		}
		if p.KeepValue(v) {
			return true
		}
	}

	return false
}

func (p *OrPredicate) KeepRange(min, max pq.Value) bool {
	for _, sub := range p.preds {
		if sub == nil || sub.KeepRange(min, max) {
			// Nil means all values are returned
			return true
		}
	}
	return false
}

type InstrumentedPredicate struct {
	Pred Predicate // Optional, if missing then just keeps metrics with no filtering
	// predicateStats holds the chunk/page counters (InspectedColumnChunks,
	// KeptColumnChunks, InspectedPages, KeptPages), incremented by the iterator's
	// keep* helpers which take &predicateStats. Promoted fields keep the public API.
	predicateStats
	InspectedValues int64
	KeptValues      int64
}

var _ Predicate = (*InstrumentedPredicate)(nil)

func (p *InstrumentedPredicate) String() string {
	if p.Pred == nil {
		return fmt.Sprintf("InstrumentedPredicate{%d, nil}", p.InspectedValues)
	}
	return fmt.Sprintf("InstrumentedPredicate{%d, %s}", p.InspectedValues, p.Pred)
}

func (p *InstrumentedPredicate) KeepValue(v pq.Value) bool {
	p.InspectedValues++

	if p.Pred == nil || p.Pred.KeepValue(v) {
		p.KeptValues++
		return true
	}

	return false
}

func (p *InstrumentedPredicate) KeepRange(min, max pq.Value) bool {
	if p.Pred == nil {
		return true
	}
	return p.Pred.KeepRange(min, max)
}

type SkipNilsPredicate struct{}

var _ Predicate = (*SkipNilsPredicate)(nil)

func NewSkipNilsPredicate() *SkipNilsPredicate {
	return &SkipNilsPredicate{}
}

func (p *SkipNilsPredicate) String() string {
	return "SkipNilsPredicate{}"
}

func (p *SkipNilsPredicate) KeepValue(v pq.Value) bool {
	return !v.IsNull()
}

func (p *SkipNilsPredicate) KeepRange(pq.Value, pq.Value) bool { return true }

type CallbackPredicate struct {
	cb func() bool
}

var _ Predicate = (*CallbackPredicate)(nil)

func NewCallbackPredicate(cb func() bool) *CallbackPredicate {
	return &CallbackPredicate{cb: cb}
}

func (m *CallbackPredicate) String() string { return "CallbackPredicate{}" }

func (m *CallbackPredicate) KeepValue(pq.Value) bool { return m.cb() }

func (m *CallbackPredicate) KeepRange(pq.Value, pq.Value) bool { return m.cb() }

var _ Predicate = (*NilValuePredicate)(nil)

type NilValuePredicate struct{}

func NewNilValuePredicate() NilValuePredicate {
	return NilValuePredicate{}
}

func (p NilValuePredicate) String() string {
	return "NilValuePredicate{}"
}

func (p NilValuePredicate) KeepValue(v pq.Value) bool {
	return v.IsNull()
}

func (p NilValuePredicate) KeepRange(pq.Value, pq.Value) bool { return false }

type IncludeNilStringEqualPredicate struct {
	value []byte
}

func NewIncludeNilStringEqualPredicate(val []byte) IncludeNilStringEqualPredicate {
	return IncludeNilStringEqualPredicate{value: val}
}

func (p IncludeNilStringEqualPredicate) String() string {
	return "IncludeNilStringEqualPredicate{}"
}

func (p IncludeNilStringEqualPredicate) KeepValue(v pq.Value) bool {
	vv := v.ByteArray()
	return bytes.Equal(vv, p.value)
}

func (p IncludeNilStringEqualPredicate) KeepRange(pq.Value, pq.Value) bool { return true }
