package parquetquery

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/grafana/tempo/pkg/regexp"

	pq "github.com/parquet-go/parquet-go"
)

// Predicate is a pushdown predicate that can be applied at
// the chunk, page, and value levels.
type Predicate interface {
	fmt.Stringer

	KeepColumnChunk(*ColumnChunkHelper) bool
	KeepPage(page pq.Page) bool
	KeepValue(pq.Value) bool
}

// StringInPredicate checks for any of the given strings.
// Case sensitive exact byte matching
type StringInPredicate struct {
	ss [][]byte
}

var _ Predicate = (*StringInPredicate)(nil)

func NewStringInPredicate(ss []string) Predicate {
	p := &StringInPredicate{
		ss: make([][]byte, len(ss)),
	}
	for i := range ss {
		p.ss[i] = []byte(ss[i])
	}
	return p
}

func (p *StringInPredicate) String() string {
	var strings string
	for i, s := range p.ss {
		if i > 0 {
			strings += ", "
		}
		strings += string(s)
	}
	return fmt.Sprintf("StringInPredicate{%s}", strings)
}

func (p *StringInPredicate) KeepColumnChunk(cc *ColumnChunkHelper) bool {
	if d := cc.Dictionary(); d != nil {
		return keepDictionary(d, p.KeepValue)
	}

	ci, err := cc.ColumnIndex()
	if err == nil && ci != nil {
		for _, subs := range p.ss {
			for i := 0; i < ci.NumPages(); i++ {
				min := ci.MinValue(i).ByteArray()
				max := ci.MaxValue(i).ByteArray()
				ok := bytes.Compare(min, subs) <= 0 && bytes.Compare(max, subs) >= 0
				if ok {
					// At least one page in this chunk matches
					return true
				}
			}
		}
		return false
	}

	return true
}

func (p *StringInPredicate) KeepValue(v pq.Value) bool {
	ba := v.ByteArray()
	for _, ss := range p.ss {
		if bytes.Equal(ba, ss) {
			return true
		}
	}
	return false
}

func (p *StringInPredicate) KeepPage(pq.Page) bool {
	// todo: check bounds
	return true
}

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

func (p *regexPredicate) KeepColumnChunk(cc *ColumnChunkHelper) bool {
	d := cc.Dictionary()

	// should we do this?
	p.matcher.Reset()

	if d != nil {
		return keepDictionary(d, p.KeepValue)
	}

	return true
}

func (p *regexPredicate) KeepValue(v pq.Value) bool {
	return p.keep(&v)
}

func (p *regexPredicate) KeepPage(pq.Page) bool {
	return true
}

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

func (p *SubstringPredicate) KeepColumnChunk(cc *ColumnChunkHelper) bool {
	// Reset match cache on each row group change
	p.matches = make(map[string]bool, len(p.matches))

	if d := cc.Dictionary(); d != nil {
		return keepDictionary(d, p.KeepValue)
	}

	return true
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

func (p *SubstringPredicate) KeepPage(pq.Page) bool {
	return true
}

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

func (p *IntBetweenPredicate) KeepColumnChunk(c *ColumnChunkHelper) bool {
	ci, err := c.ColumnIndex()
	if err == nil && ci != nil {
		for i := 0; i < ci.NumPages(); i++ {
			min := ci.MinValue(i).Int64()
			max := ci.MaxValue(i).Int64()
			if p.max >= min && p.min <= max {
				return true
			}
		}
		return false
	}

	return true
}

func (p *IntBetweenPredicate) KeepValue(v pq.Value) bool {
	vv := v.Int64()
	return p.min <= vv && vv <= p.max
}

func (p *IntBetweenPredicate) KeepPage(page pq.Page) bool {
	if min, max, ok := page.Bounds(); ok {
		return p.max >= min.Int64() && p.min <= max.Int64()
	}
	return true
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

func (p *GenericPredicate[T]) KeepColumnChunk(c *ColumnChunkHelper) bool {
	if d := c.Dictionary(); d != nil {
		return keepDictionary(d, p.KeepValue)
	}

	if p.RangeFn == nil {
		return true
	}

	ci, err := c.ColumnIndex()
	if err == nil && ci != nil {
		for i := 0; i < ci.NumPages(); i++ {
			min := p.Extract(ci.MinValue(i))
			max := p.Extract(ci.MaxValue(i))
			if p.RangeFn(min, max) {
				return true
			}
		}
		return false
	}

	return true
}

func (p *GenericPredicate[T]) KeepPage(page pq.Page) bool {
	if p.RangeFn != nil {
		if min, max, ok := page.Bounds(); ok {
			return p.RangeFn(p.Extract(min), p.Extract(max))
		}
	}

	return true
}

func (p *GenericPredicate[T]) KeepValue(v pq.Value) bool {
	return p.Fn(p.Extract(v))
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

func (p *OrPredicate) KeepColumnChunk(c *ColumnChunkHelper) bool {
	ret := false
	for _, p := range p.preds {
		if p == nil {
			// Nil means all values are returned
			ret = ret || true
			continue
		}
		if p.KeepColumnChunk(c) {
			ret = ret || true
		}
	}

	return ret
}

func (p *OrPredicate) KeepPage(page pq.Page) bool {
	for _, p := range p.preds {
		if p == nil {
			// Nil means all values are returned
			return true
		}
		if p.KeepPage(page) {
			return true
		}
	}

	return false
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

type InstrumentedPredicate struct {
	Pred                  Predicate // Optional, if missing then just keeps metrics with no filtering
	InspectedColumnChunks int64
	InspectedPages        int64
	InspectedValues       int64
	KeptColumnChunks      int64
	KeptPages             int64
	KeptValues            int64
}

var _ Predicate = (*InstrumentedPredicate)(nil)

func (p *InstrumentedPredicate) String() string {
	if p.Pred == nil {
		return fmt.Sprintf("InstrumentedPredicate{%d, nil}", p.InspectedValues)
	}
	return fmt.Sprintf("InstrumentedPredicate{%d, %s}", p.InspectedValues, p.Pred)
}

func (p *InstrumentedPredicate) KeepColumnChunk(c *ColumnChunkHelper) bool {
	p.InspectedColumnChunks++

	if p.Pred == nil || p.Pred.KeepColumnChunk(c) {
		p.KeptColumnChunks++
		return true
	}

	return false
}

func (p *InstrumentedPredicate) KeepPage(page pq.Page) bool {
	p.InspectedPages++

	if p.Pred == nil || p.Pred.KeepPage(page) {
		p.KeptPages++
		return true
	}

	return false
}

func (p *InstrumentedPredicate) KeepValue(v pq.Value) bool {
	p.InspectedValues++

	if p.Pred == nil || p.Pred.KeepValue(v) {
		p.KeptValues++
		return true
	}

	return false
}

// keepDictionary inspects all values using the callback and returns if any
// matches were found.
func keepDictionary(dict pq.Dictionary, keepValue func(pq.Value) bool) bool {
	l := dict.Len()
	for i := 0; i < l; i++ {
		dictionaryEntry := dict.Index(int32(i))
		if keepValue(dictionaryEntry) {
			return true
		}
	}

	return false
}

type SkipNilsPredicate struct{}

var _ Predicate = (*SkipNilsPredicate)(nil)

func NewSkipNilsPredicate() *SkipNilsPredicate {
	return &SkipNilsPredicate{}
}

func (p *SkipNilsPredicate) String() string {
	return "SkipNilsPredicate{}"
}

func (p *SkipNilsPredicate) KeepColumnChunk(*ColumnChunkHelper) bool {
	return true
}

func (p *SkipNilsPredicate) KeepPage(page pq.Page) bool {
	return page.NumValues() > page.NumNulls()
}

func (p *SkipNilsPredicate) KeepValue(v pq.Value) bool {
	return !v.IsNull()
}

type CallbackPredicate struct {
	cb func() bool
}

var _ Predicate = (*CallbackPredicate)(nil)

func NewCallbackPredicate(cb func() bool) *CallbackPredicate {
	return &CallbackPredicate{cb: cb}
}

func (m *CallbackPredicate) String() string { return "CallbackPredicate{}" }

func (m *CallbackPredicate) KeepColumnChunk(*ColumnChunkHelper) bool { return m.cb() }

func (m *CallbackPredicate) KeepPage(pq.Page) bool { return m.cb() }

func (m *CallbackPredicate) KeepValue(pq.Value) bool { return m.cb() }
