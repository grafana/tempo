package parquetquery

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

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
				ok := bytes.Compare(ci.MinValue(i).ByteArray(), subs) <= 0 && bytes.Compare(ci.MaxValue(i).ByteArray(), subs) >= 0
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
	regs        []*regexp.Regexp
	matches     map[string]bool
	shouldMatch bool
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
	p := &regexPredicate{
		regs:        make([]*regexp.Regexp, 0, len(regs)),
		matches:     make(map[string]bool),
		shouldMatch: shouldMatch,
	}
	for _, reg := range regs {
		r, err := regexp.Compile(reg)
		if err != nil {
			return nil, err
		}
		p.regs = append(p.regs, r)
	}
	return p, nil
}

func (p *regexPredicate) String() string {
	var strings string
	for _, s := range p.regs {
		strings += fmt.Sprintf("%s, ", s.String())
	}
	return fmt.Sprintf("RegexInPredicate{%s}", strings)
}

func (p *regexPredicate) keep(v *pq.Value) bool {
	if v.IsNull() {
		return false
	}

	b := v.ByteArray()

	// Check uses zero alloc optimization of map[string([]byte)]
	if matched, ok := p.matches[string(b)]; ok {
		return matched
	}

	matched := false
	for _, r := range p.regs {
		if r.Match(b) == p.shouldMatch {
			matched = true
			break
		}
	}

	// Only alloc the string when updating the map
	p.matches[string(b)] = matched

	return matched
}

func (p *regexPredicate) KeepColumnChunk(cc *ColumnChunkHelper) bool {
	d := cc.Dictionary()

	// Reset match cache on each row group change
	// Use exact size of the incoming dictionary
	// if present and larger.
	count := len(p.matches)
	if d != nil && d.Len() > count {
		count = d.Len()
	}
	p.matches = make(map[string]bool, count)

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

func NewIntPredicate(fn func(int64) bool, rangeFn func(int64, int64) bool) *GenericPredicate[int64] {
	return NewGenericPredicate(
		fn, rangeFn,
		func(v pq.Value) int64 { return v.Int64() },
	)
}

func NewFloatPredicate(fn func(float64) bool, rangeFn func(float64, float64) bool) *GenericPredicate[float64] {
	return NewGenericPredicate(
		fn, rangeFn,
		func(v pq.Value) float64 { return v.Double() },
	)
}

func NewBoolPredicate(b bool) *GenericPredicate[bool] {
	return NewGenericPredicate(
		func(v bool) bool { return v == b },
		nil,
		func(v pq.Value) bool { return v.Boolean() },
	)
}

type FloatBetweenPredicate struct {
	min, max float64
}

var _ Predicate = (*FloatBetweenPredicate)(nil)

func NewFloatBetweenPredicate(min, max float64) *FloatBetweenPredicate {
	return &FloatBetweenPredicate{min, max}
}

func (p *FloatBetweenPredicate) String() string {
	return fmt.Sprintf("FloatBetweenPredicate{%f,%f}", p.min, p.max)
}

func (p *FloatBetweenPredicate) KeepColumnChunk(c *ColumnChunkHelper) bool {
	ci, err := c.ColumnIndex()
	if err == nil && ci != nil {
		for i := 0; i < ci.NumPages(); i++ {
			min := ci.MinValue(i).Double()
			max := ci.MaxValue(i).Double()
			if p.max >= min && p.min <= max {
				return true
			}
		}
		return false
	}

	return true
}

func (p *FloatBetweenPredicate) KeepValue(v pq.Value) bool {
	vv := v.Double()
	return p.min <= vv && vv <= p.max
}

func (p *FloatBetweenPredicate) KeepPage(page pq.Page) bool {
	if min, max, ok := page.Bounds(); ok {
		return p.max >= min.Double() && p.min <= max.Double()
	}
	return true
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
	pred                  Predicate // Optional, if missing then just keeps metrics with no filtering
	InspectedColumnChunks int64
	InspectedPages        int64
	InspectedValues       int64
	KeptColumnChunks      int64
	KeptPages             int64
	KeptValues            int64
}

var _ Predicate = (*InstrumentedPredicate)(nil)

func (p *InstrumentedPredicate) String() string {
	if p.pred == nil {
		return fmt.Sprintf("InstrumentedPredicate{%d, nil}", p.InspectedValues)
	}
	return fmt.Sprintf("InstrumentedPredicate{%d, %s}", p.InspectedValues, p.pred)
}

func (p *InstrumentedPredicate) KeepColumnChunk(c *ColumnChunkHelper) bool {
	p.InspectedColumnChunks++

	if p.pred == nil || p.pred.KeepColumnChunk(c) {
		p.KeptColumnChunks++
		return true
	}

	return false
}

func (p *InstrumentedPredicate) KeepPage(page pq.Page) bool {
	p.InspectedPages++

	if p.pred == nil || p.pred.KeepPage(page) {
		p.KeptPages++
		return true
	}

	return false
}

func (p *InstrumentedPredicate) KeepValue(v pq.Value) bool {
	p.InspectedValues++

	if p.pred == nil || p.pred.KeepValue(v) {
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
