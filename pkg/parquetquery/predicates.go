package parquetquery

import (
	"bytes"
	"strings"

	pq "github.com/segmentio/parquet-go"
	"github.com/uber-go/atomic"
)

// parquetPredicate is a pushdown predicate that can be applied at
// the chunk, page, and value levels.
type Predicate interface {
	KeepColumnChunk(cc pq.ColumnChunk) bool
	KeepPage(page pq.Page) bool
	KeepValue(pq.Value) bool
}

// StringPredicate checks for exact string match.
type StringPredicate struct {
	s []byte
}

var _ Predicate = (*StringPredicate)(nil)

func NewStringPredicate(s string) Predicate {
	return &StringPredicate{
		s: []byte(s),
	}
}

func (s *StringPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
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

func (s *StringPredicate) KeepValue(v pq.Value) bool {
	return bytes.Equal(v.ByteArray(), s.s)
}

func (s *StringPredicate) KeepPage(page pq.Page) bool {
	// todo: check bounds

	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict != nil && dict.Len() > 0 {
		len := dict.Len()
		found := false

		for i := 0; i < len; i++ {
			if bytes.Equal(dict.Index(int32(i)).ByteArray(), s.s) {
				found = true
				break
			}
		}

		return found
	}

	return true
}

// stringPredicate checks for exact string match.
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

func (s *StringInPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
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

func (s *StringInPredicate) KeepValue(v pq.Value) bool {
	ba := v.ByteArray()
	for _, ss := range s.ss {
		if bytes.Equal(ba, ss) {
			return true
		}
	}
	return false
}

func (s *StringInPredicate) KeepPage(page pq.Page) bool {
	// todo: check bounds

	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict != nil && dict.Len() > 0 {
		len := dict.Len()

		for i := 0; i < len; i++ {
			dictionaryEntry := dict.Index(int32(i)).ByteArray()
			for _, subs := range s.ss {
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

type SubstringPredicate struct {
	substring string
	matches   map[string]bool
}

var _ Predicate = (*SubstringPredicate)(nil)

func NewSubstringPredicate(substring string) *SubstringPredicate {
	return &SubstringPredicate{
		substring: strings.ToLower(substring),
		matches:   map[string]bool{},
	}
}

func (s *SubstringPredicate) KeepColumnChunk(_ pq.ColumnChunk) bool {
	// Is there any filtering possible here?
	// Column chunk contains a bloom filter and min/max bounds,
	// but those can't be inspected for a substring match.
	return true
}

func (s *SubstringPredicate) KeepValue(v pq.Value) bool {
	vs := v.String()
	if m, ok := s.matches[vs]; ok {
		return m
	}

	m := strings.Contains(strings.ToLower(vs), s.substring)
	s.matches[vs] = m
	return m
	//return strings.Contains(strings.ToLower(v.String()), s.substring)
}

func (s *SubstringPredicate) KeepPage(page pq.Page) bool {
	// Reset match cache on each page change
	s.matches = make(map[string]bool, len(s.matches))

	// If a dictionary column then ensure at least one matching
	// value exists in the dictionary
	dict := page.Dictionary()
	if dict != nil && dict.Len() > 0 {
		len := dict.Len()
		for i := 0; i < len; i++ {
			if s.KeepValue(dict.Index(int32(i))) {
				return true
			}
		}

		return false
	}

	return true
}

type PrefixPredicate struct {
	prefix []byte
}

var _ Predicate = (*PrefixPredicate)(nil)

func NewPrefixPredicate(prefix string) Predicate {
	return &PrefixPredicate{
		prefix: []byte(prefix),
	}
}

func (s *PrefixPredicate) KeepColumnChunk(cc pq.ColumnChunk) bool {
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

func (s *PrefixPredicate) KeepValue(v pq.Value) bool {
	return bytes.HasPrefix(v.ByteArray(), s.prefix)
}

func (d *PrefixPredicate) KeepPage(page pq.Page) bool {
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

// IntBetweenPredicate checks for int between the bounds [min,max] inclusive
type IntBetweenPredicate struct {
	min, max int64
}

var _ Predicate = (*IntBetweenPredicate)(nil)

func NewIntBetweenPredicate(min, max int64) *IntBetweenPredicate {
	return &IntBetweenPredicate{min, max}
}

func (s *IntBetweenPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
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

func (s *IntBetweenPredicate) KeepValue(v pq.Value) bool {
	vv := v.Int64()
	return s.min <= vv && vv <= s.max
}

func (s *IntBetweenPredicate) KeepPage(page pq.Page) bool {
	if min, max, ok := page.Bounds(); ok {
		return s.max >= min.Int64() && s.min <= max.Int64()
	}
	return true
}

type OrPredicate struct {
	predicates []Predicate
}

var _ Predicate = (*OrPredicate)(nil)

func NewOrPredicate(preds ...Predicate) *OrPredicate {
	return &OrPredicate{
		predicates: preds,
	}
}

func (s *OrPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
	for _, p := range s.predicates {
		if p.KeepColumnChunk(c) {
			return true
		}
	}

	return false
}

func (s *OrPredicate) KeepValue(v pq.Value) bool {
	for _, p := range s.predicates {
		if p.KeepValue(v) {
			return true
		}
	}
	return false
}

func (s *OrPredicate) KeepPage(page pq.Page) bool {
	for _, p := range s.predicates {
		if p.KeepPage(page) {
			return true
		}
	}
	return false
}

type InstrumentedPredicate struct {
	pred                  Predicate // Optional, if missing then just keeps metrics with no filtering
	InspectedColumnChunks atomic.Int64
	InspectedPages        atomic.Int64
	InspectedValues       atomic.Int64
	KeptColumnChunks      atomic.Int64
	KeptPages             atomic.Int64
	KeptValues            atomic.Int64
}

var _ Predicate = (*InstrumentedPredicate)(nil)

func (s *InstrumentedPredicate) KeepColumnChunk(c pq.ColumnChunk) bool {
	s.InspectedColumnChunks.Inc()
	if s.pred == nil {
		return true
	}

	if s.pred.KeepColumnChunk(c) {
		s.KeptColumnChunks.Inc()
		return true
	}

	return false
}

func (s *InstrumentedPredicate) KeepPage(page pq.Page) bool {
	s.InspectedPages.Inc()
	if s.pred == nil {
		return true
	}

	if s.pred.KeepPage(page) {
		s.KeptPages.Inc()
		return true
	}

	return false
}

func (s *InstrumentedPredicate) KeepValue(v pq.Value) bool {
	s.InspectedValues.Inc()
	if s.pred == nil {
		return true
	}

	if s.pred.KeepValue(v) {
		s.KeptValues.Inc()
		return true
	}

	return false
}
