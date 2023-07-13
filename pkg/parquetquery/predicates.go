package parquetquery

import (
	"bytes"
	"fmt"
	"regexp"

	pq "github.com/segmentio/parquet-go"
)

// Predicate is a pushdown predicate that can be applied at
// the chunk, page, and value levels.
type Predicate interface {
	fmt.Stringer

	KeepColumnChunk(cc pq.ColumnChunk) (bool, pq.Pages, pq.Page)
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

func (p *StringInPredicate) KeepColumnChunk(cc pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	p.helper.setNewRowGroup()

	if ci := cc.ColumnIndex(); ci != nil {
		for _, subs := range p.ss {
			for i := 0; i < ci.NumPages(); i++ {
				ok := bytes.Compare(ci.MinValue(i).ByteArray(), subs) <= 0 && bytes.Compare(ci.MaxValue(i).ByteArray(), subs) >= 0
				if ok {
					// At least one page in this chunk matches
					return true, nil, nil
				}
			}
		}
		return false, nil, nil
	}

	// Check the row group dictionary (accessible through the first page)
	// If present then use it to skip row group or not.
	pgs := cc.Pages()

	firstPage, _ := pgs.ReadPage()
	if firstPage == nil {
		// Failed to read the first page, can't make a determination
		pgs.Close()
		return true, nil, nil
	}

	if firstPage.Dictionary() == nil {
		// Not a dictionary column
		// TODO - Can we check this earlier?
		pgs.Close()
		pq.Release(firstPage)
		return true, nil, nil
	}

	if !p.helper.keepPage(firstPage, p.KeepValue) {
		// No match, cleanup and skip
		pq.Release(firstPage)
		pgs.Close()
		return false, nil, nil
	}

	return true, pgs, firstPage
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

type regexPredicate struct {
	regs        []*regexp.Regexp
	matches     map[string]bool
	shouldMatch bool

	helper DictionaryPredicateHelper
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

func (p *regexPredicate) KeepColumnChunk(cc pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	p.helper.setNewRowGroup()

	// Reset match cache on each row group change
	p.matches = make(map[string]bool, len(p.matches))

	// Check the row group dictionary (accessible through the first page)
	// If present then use it to skip row group or not.
	pgs := cc.Pages()

	firstPage, _ := pgs.ReadPage()
	if firstPage == nil {
		// Failed to read the first page, can't make a determination
		pgs.Close()
		return true, nil, nil
	}

	if firstPage.Dictionary() == nil {
		// Not a dictionary column
		// TODO - Can we check this earlier?
		pgs.Close()
		pq.Release(firstPage)
		return true, nil, nil
	}

	if !p.helper.keepPage(firstPage, p.KeepValue) {
		// No match, cleanup and skip
		pq.Release(firstPage)
		pgs.Close()
		return false, nil, nil
	}

	return true, pgs, firstPage
}

func (p *regexPredicate) KeepValue(v pq.Value) bool {
	return p.keep(&v)
}

func (p *regexPredicate) KeepPage(page pq.Page) bool {
	if p.helper.newRowGroup {
		// Reset match cache on each row group change
		// We delay until the first page is received
		// so we can get an accurate count of the number
		// of distinct values for dictionary columns.
		count := len(p.matches)
		if d := page.Dictionary(); d != nil {
			if d.Len() > count {
				count = d.Len()
			}
		}
		p.matches = make(map[string]bool, count)
	}

	return p.helper.keepPage(page, p.KeepValue)
}

type SubstringPredicate struct {
	substring []byte
	matches   map[string]bool

	helper DictionaryPredicateHelper
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

func (p *SubstringPredicate) KeepColumnChunk(cc pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	p.helper.setNewRowGroup()

	// Reset match cache on each row group change
	p.matches = make(map[string]bool, len(p.matches))

	// Check the row group dictionary (accessible through the first page)
	// If present then use it to skip row group or not.
	pgs := cc.Pages()

	firstPage, _ := pgs.ReadPage()
	if firstPage == nil {
		// Failed to read the first page, can't make a determination
		pgs.Close()
		return true, nil, nil
	}

	if firstPage.Dictionary() == nil {
		// Not a dictionary column
		// TODO - Can we check this earlier?
		pgs.Close()
		pq.Release(firstPage)
		return true, nil, nil
	}

	if !p.helper.keepPage(firstPage, p.KeepValue) {
		// No match, cleanup and skip
		pq.Release(firstPage)
		pgs.Close()
		return false, nil, nil
	}

	return true, pgs, firstPage
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

func (p *IntBetweenPredicate) KeepColumnChunk(c pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	if ci := c.ColumnIndex(); ci != nil {
		for i := 0; i < ci.NumPages(); i++ {
			min := ci.MinValue(i).Int64()
			max := ci.MaxValue(i).Int64()
			if p.max >= min && p.min <= max {
				return true, nil, nil
			}
		}
		return false, nil, nil
	}

	return true, nil, nil
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
	return "GenericPredicate{}"
}

func (p *GenericPredicate[T]) KeepColumnChunk(c pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	p.helper.setNewRowGroup()

	if p.RangeFn == nil {
		return true, nil, nil
	}

	if ci := c.ColumnIndex(); ci != nil {
		for i := 0; i < ci.NumPages(); i++ {
			min := p.Extract(ci.MinValue(i))
			max := p.Extract(ci.MaxValue(i))
			if p.RangeFn(min, max) {
				return true, nil, nil
			}
		}
		return false, nil, nil
	}

	return true, nil, nil
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

func (p *FloatBetweenPredicate) KeepColumnChunk(c pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	if ci := c.ColumnIndex(); ci != nil {
		for i := 0; i < ci.NumPages(); i++ {
			min := ci.MinValue(i).Double()
			max := ci.MaxValue(i).Double()
			if p.max >= min && p.min <= max {
				return true, nil, nil
			}
		}
		return false, nil, nil
	}

	return true, nil, nil
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
		if pred != nil {
			preds += pred.String() + ","
		} else {
			preds += "nil,"
		}
	}
	return fmt.Sprintf("OrPredicate{%s}", preds)
}

func (p *OrPredicate) KeepColumnChunk(c pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	ret := false
	var pgs pq.Pages
	var pg pq.Page
	var keep bool
	for _, p := range p.preds {
		if p == nil {
			// Nil means all values are returned
			ret = ret || true
			continue
		}
		keep, pgs, pg = p.KeepColumnChunk(c)
		if keep {
			ret = ret || true
		}
	}

	return ret, pgs, pg
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

func (p *InstrumentedPredicate) KeepColumnChunk(c pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	p.InspectedColumnChunks++

	if p.pred == nil {
		p.KeptColumnChunks++
		return true, nil, nil
	}

	if keep, pgs, pg := p.pred.KeepColumnChunk(c); keep {
		p.KeptColumnChunks++
		return true, pgs, pg
	}

	return false, nil, nil
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

type SkipNilsPredicate struct{}

var _ Predicate = (*SkipNilsPredicate)(nil)

func NewSkipNilsPredicate() *SkipNilsPredicate {
	return &SkipNilsPredicate{}
}

func (p *SkipNilsPredicate) String() string {
	return "SkipNilsPredicate{}"
}

func (p *SkipNilsPredicate) KeepColumnChunk(pq.ColumnChunk) (bool, pq.Pages, pq.Page) {
	return true, nil, nil
}

func (p *SkipNilsPredicate) KeepPage(page pq.Page) bool {
	return page.NumValues() > page.NumNulls()
}

func (p *SkipNilsPredicate) KeepValue(v pq.Value) bool {
	return !v.IsNull()
}
