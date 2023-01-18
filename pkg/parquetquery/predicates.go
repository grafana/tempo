package parquetquery

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	pq "github.com/segmentio/parquet-go"
)

// Predicate is a pushdown predicate that can be applied at
// the chunk, page, and value levels.
type Predicate interface {
	fmt.Stringer

	KeepColumnChunk(cc pq.ColumnChunk) bool
	KeepPage(page pq.Page) bool
	KeepValue(pq.Value) bool
}

// StringInPredicate checks for any of the given strings.
// Case sensitive exact byte matching
type StringInPredicate struct {
	ss [][]byte

	helper DictionaryPredicateHelper
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
	for _, s := range p.ss {
		strings += fmt.Sprintf("%s, ", string(s))
	}
	return fmt.Sprintf("StringInPredicate{%s}", strings)
}

func (p *StringInPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
	p.helper.setNewRowGroup()

	if ci := cc.ColumnIndex(); ci != nil {
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

func (p *StringInPredicate) KeepPage(page pq.Page) bool {
	//

	// todo: check bounds
	return p.helper.keepPage(page, p.KeepValue)
}

// RegexInPredicate checks for match against any of the given regexs.
// Memoized and resets on each row group.
type RegexInPredicate struct {
	regs    []*regexp.Regexp
	matches map[string]bool

	helper DictionaryPredicateHelper
}

var _ Predicate = (*RegexInPredicate)(nil)

func NewRegexInPredicate(regs []string) (*RegexInPredicate, error) {
	p := &RegexInPredicate{
		regs:    make([]*regexp.Regexp, 0, len(regs)),
		matches: make(map[string]bool),
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

func (p *RegexInPredicate) String() string {
	var strings string
	for _, s := range p.regs {
		strings += fmt.Sprintf("%s, ", s.String())
	}
	return fmt.Sprintf("RegexInPredicate{%s}", strings)
}

func (p *RegexInPredicate) keep(v *pq.Value) bool {
	if v.IsNull() {
		// Null
		return false
	}

	s := v.String()
	if matched, ok := p.matches[s]; ok {
		return matched
	}

	matched := false
	for _, r := range p.regs {
		if r.MatchString(s) {
			matched = true
			break
		}
	}

	p.matches[s] = matched
	return matched
}

func (p *RegexInPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
	p.helper.setNewRowGroup()

	// Reset match cache on each row group change
	p.matches = make(map[string]bool, len(p.matches))

	// Can we do any filtering here?
	return true
}

func (p *RegexInPredicate) KeepValue(v pq.Value) bool {
	return p.keep(&v)
}

func (p *RegexInPredicate) KeepPage(page pq.Page) bool {
	return p.helper.keepPage(page, p.KeepValue)
}

type SubstringPredicate struct {
	substring string
	matches   map[string]bool

	helper DictionaryPredicateHelper
}

var _ Predicate = (*SubstringPredicate)(nil)

func NewSubstringPredicate(substring string) *SubstringPredicate {
	return &SubstringPredicate{
		substring: substring,
		matches:   map[string]bool{},
	}
}

func (p *SubstringPredicate) String() string {
	return fmt.Sprintf("SubstringPredicate{%s}", p.substring)
}

func (p *SubstringPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
	p.helper.setNewRowGroup()

	// Reset match cache on each row group change
	p.matches = make(map[string]bool, len(p.matches))

	// Is there any filtering possible here?
	// Column chunk contains a bloom filter and min/max bounds,
	// but those can't be inspected for a substring match.
	return true
}

func (p *SubstringPredicate) KeepValue(v pq.Value) bool {
	vs := v.String()
	if m, ok := p.matches[vs]; ok {
		return m
	}

	m := strings.Contains(vs, p.substring)
	p.matches[vs] = m
	return m
}

func (p *SubstringPredicate) KeepPage(page pq.Page) bool {
	return p.helper.keepPage(page, p.KeepValue)
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

func (p *IntBetweenPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {

	if ci := c.ColumnIndex(); ci != nil {
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

// Generic predicate with callbacks to evalulate data of type T
// Fn evalulates a single data point and is required. Optionally,
// a RangeFn can evalulate a min/max range and is used to
// skip column chunks and pages when RangeFn is supplied and
// the column chunk or page also include bounds metadata.
type GenericPredicate[T any] struct {
	Fn      func(T) bool
	RangeFn func(min, max T) bool
	Extract func(pq.Value) T

	helper DictionaryPredicateHelper
}

var _ Predicate = (*GenericPredicate[int64])(nil)

func NewGenericPredicate[T any](fn func(T) bool, rangeFn func(T, T) bool, extract func(pq.Value) T) *GenericPredicate[T] {
	return &GenericPredicate[T]{Fn: fn, RangeFn: rangeFn, Extract: extract}
}

func (p *GenericPredicate[T]) String() string {
	return fmt.Sprintf("GenericPredicate{}")
}

func (p *GenericPredicate[T]) KeepColumnChunk(c pq.ColumnChunk) bool {
	p.helper.setNewRowGroup()

	if p.RangeFn == nil {
		return true
	}

	if ci := c.ColumnIndex(); ci != nil {
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

	return p.helper.keepPage(page, p.KeepValue)
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

func (p *FloatBetweenPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {

	if ci := c.ColumnIndex(); ci != nil {
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
	var preds string
	for _, pred := range p.preds {
		preds += pred.String() + ","
	}
	return fmt.Sprintf("OrPredicate{%s}", p.preds)
}

func (p *OrPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
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

func (p *InstrumentedPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
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

// DictionaryPredicateHelper is a helper for a predicate that uses a dictionary
// for filtering.
//
// There is one dictionary per ColumnChunk/RowGroup, but it is not accessible in
// KeepColumnChunk. This helper saves the result of KeepPage and uses it for
// all pages in the row group. It also has a basic heuristic for choosing not
// to check the dictionary at all if the cardinality is too high.
type DictionaryPredicateHelper struct {
	newRowGroup         bool
	keepPagesInRowGroup bool
}

func (d *DictionaryPredicateHelper) setNewRowGroup() {
	d.newRowGroup = true
}

func (d *DictionaryPredicateHelper) keepPage(page pq.Page, keepValue func(pq.Value) bool) bool {
	if !d.newRowGroup {
		return d.keepPagesInRowGroup
	}

	d.newRowGroup = false
	d.keepPagesInRowGroup = true

	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict == nil {
		return d.keepPagesInRowGroup
	}

	l := dict.Len()
	d.keepPagesInRowGroup = false
	for i := 0; i < l; i++ {
		dictionaryEntry := dict.Index(int32(i))
		if keepValue(dictionaryEntry) {
			d.keepPagesInRowGroup = true
			break
		}
	}

	return d.keepPagesInRowGroup
}

type SkipNilsPredicate struct {
}

var _ Predicate = (*SkipNilsPredicate)(nil)

func NewSkipNilsPredicate() *SkipNilsPredicate {
	return &SkipNilsPredicate{}
}

func (p *SkipNilsPredicate) String() string {
	return "SkipNilsPredicate{}"
}

func (p *SkipNilsPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
	return true
}

func (p *SkipNilsPredicate) KeepPage(page pq.Page) bool {
	return page.NumValues() > page.NumNulls()
}

func (p *SkipNilsPredicate) KeepValue(v pq.Value) bool {
	return !v.IsNull()
}
