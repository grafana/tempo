package parquetquery

import (
	"bytes"
	"strings"

	pq "github.com/segmentio/parquet-go"
)

// parquetPredicate is a pushdown predicate that can be applied at
// the chunk, page, and value levels.
type Predicate interface {
	KeepColumnChunk(cc pq.ColumnChunk) bool
	KeepPage(page pq.Page) bool
	KeepValue(pq.Value) bool
}

// stringPredicate checks for exact string match.
type stringPredicate struct {
	s []byte
}

var _ Predicate = (*stringPredicate)(nil)

func NewStringPredicate(s string) Predicate {
	return &stringPredicate{
		s: []byte(s),
	}
}

func (s *stringPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
	ci := cc.ColumnIndex()

	for i := 0; i < ci.NumPages(); i++ {
		ok := bytes.Compare(ci.MinValue(i).ByteArray(), s.s) <= 0 && bytes.Compare(ci.MaxValue(i).ByteArray(), s.s) >= 0
		if ok {
			// At least one page in this chunk matches
			return true
		}
	}

	return false
}

func (s *stringPredicate) KeepValue(v pq.Value) bool {
	return bytes.Equal(v.ByteArray(), s.s)
}

func (d *stringPredicate) KeepPage(page pq.Page) bool {
	// todo: check bounds

	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict != nil && dict.Len() > 0 {
		len := dict.Len()
		found := false

		for i := 0; i < len; i++ {
			if bytes.Equal(dict.Index(int32(i)).ByteArray(), d.s) {
				found = true
				break
			}
		}

		return found
	}

	return true
}

// stringPredicate checks for exact string match.
type stringInPredicate struct {
	ss [][]byte
}

var _ Predicate = (*stringInPredicate)(nil)

func NewStringInPredicate(ss []string) Predicate {
	p := &stringInPredicate{
		ss: make([][]byte, len(ss)),
	}
	for i := range ss {
		p.ss[i] = []byte(ss[i])
	}
	return p
}

func (s *stringInPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
	ci := cc.ColumnIndex()

	for _, subs := range s.ss {
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

func (s *stringInPredicate) KeepValue(v pq.Value) bool {
	ba := v.ByteArray()
	for _, ss := range s.ss {
		if bytes.Equal(ba, ss) {
			return true
		}
	}
	return false
}

func (d *stringInPredicate) KeepPage(page pq.Page) bool {
	// todo: check bounds

	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict != nil && dict.Len() > 0 {
		len := dict.Len()

		for i := 0; i < len; i++ {
			dictionaryEntry := dict.Index(int32(i)).ByteArray()
			for _, subs := range d.ss {
				if bytes.Equal(dictionaryEntry, subs) {
					// At least 1 string present in this page
					return true
				}
			}
		}

		return false
	}

	return true
}

type substringPredicate struct {
	substring string
}

var _ Predicate = (*substringPredicate)(nil)

func NewSubstringPredicate(substring string) Predicate {
	return &substringPredicate{
		substring: strings.ToLower(substring),
	}
}

func (s *substringPredicate) KeepColumnChunk(_ pq.ColumnChunk) bool {
	// Is there any filtering possible here?
	// Column chunk contains a bloom filter and min/max bounds,
	// but those can't be inspected for a substring match.
	return true
}

func (s *substringPredicate) check(ss string) bool {
	return strings.Contains(strings.ToLower(ss), s.substring)
}

func (s *substringPredicate) KeepValue(v pq.Value) bool {
	return s.check(v.String())
}

func (s *substringPredicate) KeepPage(page pq.Page) bool {
	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict != nil && dict.Len() > 0 {
		len := dict.Len()
		found := false

		for i := 0; i < len; i++ {
			if s.check(dict.Index(int32(i)).String()) {
				found = true
				break
			}
		}

		return found
	}

	return true
}

type prefixPredicate struct {
	prefix []byte
}

var _ Predicate = (*prefixPredicate)(nil)

func NewPrefixPredicate(prefix string) Predicate {
	return &prefixPredicate{
		prefix: []byte(prefix),
	}
}

func (s *prefixPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
	ci := cc.ColumnIndex()

	for i := 0; i < ci.NumPages(); i++ {
		ok := bytes.Compare(ci.MinValue(i).ByteArray(), s.prefix) <= 0 && bytes.Compare(ci.MaxValue(i).ByteArray(), s.prefix) >= 0
		if ok {
			// At least one page in this chunk matches
			return true
		}
	}

	return false
}

func (s *prefixPredicate) KeepValue(v pq.Value) bool {
	return bytes.HasPrefix(v.ByteArray(), s.prefix)
}

func (d *prefixPredicate) KeepPage(page pq.Page) bool {
	// Check bounds
	if min, max, ok := page.Bounds(); ok {
		if bytes.Compare(min.ByteArray(), d.prefix) == 1 || bytes.Compare(d.prefix, max.ByteArray()) == -1 {
			return false
		}
	}

	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict != nil && dict.Len() > 0 {
		len := dict.Len()
		found := false

		for i := 0; i < len; i++ {
			if bytes.HasPrefix(dict.Index(int32(i)).ByteArray(), d.prefix) {
				found = true
				break
			}
		}

		return found
	}

	return true
}

// intBetweenPredicate checks for int between the bounds [min,max] inclusive
type intBetweenPredicate struct {
	min, max int64
}

var _ Predicate = (*intBetweenPredicate)(nil)

func NewIntBetweenPredicate(min, max int64) *intBetweenPredicate {
	return &intBetweenPredicate{min, max}
}

func (s *intBetweenPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
	ci := c.ColumnIndex()

	for i := 0; i < ci.NumPages(); i++ {
		min := ci.MinValue(i).Int64()
		max := ci.MaxValue(i).Int64()
		if s.max >= min && s.min <= max {
			return true
		}
	}

	return false
}

func (s *intBetweenPredicate) KeepValue(v pq.Value) bool {
	vv := v.Int64()
	return s.min <= vv && vv <= s.max
}

func (s *intBetweenPredicate) KeepPage(page pq.Page) bool {
	if min, max, ok := page.Bounds(); ok {
		return s.max >= min.Int64() && s.min <= max.Int64()
	}
	return true
}

type orPredicate struct {
	predicates []Predicate
}

var _ Predicate = (*orPredicate)(nil)

func NewOrPredicate(preds ...Predicate) *orPredicate {
	return &orPredicate{
		predicates: preds,
	}
}

func (s *orPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
	for _, p := range s.predicates {
		if p.KeepColumnChunk(c) {
			return true
		}
	}

	return false
}

func (s *orPredicate) KeepValue(v pq.Value) bool {
	for _, p := range s.predicates {
		if p.KeepValue(v) {
			return true
		}
	}
	return false
}

func (s *orPredicate) KeepPage(page pq.Page) bool {
	for _, p := range s.predicates {
		if p.KeepPage(page) {
			return true
		}
	}
	return false
}

type InstrumentedPredicate struct {
	pred                  Predicate // Optional, if missing then just keeps metrics with no filtering
	InspectedColumnChunks int
	InspectedPages        int
	InspectedValues       int
	KeptColumnChunks      int
	KeptPages             int
	KeptValues            int
}

var _ Predicate = (*InstrumentedPredicate)(nil)

func (s *InstrumentedPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
	s.InspectedColumnChunks++
	if s.pred == nil {
		return true
	}

	if s.pred.KeepColumnChunk(c) {
		s.KeptColumnChunks++
		return true
	}

	return false
}

func (s *InstrumentedPredicate) KeepPage(page pq.Page) bool {
	s.InspectedPages++
	if s.pred == nil {
		return true
	}

	if s.pred.KeepPage(page) {
		s.KeptPages++
		return true
	}

	return false
}

func (s *InstrumentedPredicate) KeepValue(v pq.Value) bool {
	s.InspectedValues++
	if s.pred == nil {
		return true
	}

	if s.pred.KeepValue(v) {
		s.KeptValues++
		return true
	}

	return false
}
