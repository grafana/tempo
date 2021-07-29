package search

import (
	"math"
	"sort"
	"time"

	"github.com/grafana/tempo/pkg/tempofb"
)

type CacheEntry struct {
	values map[string]int64 // value -> unix timestamp
}

const maxValuesPerTag = 50

type SearchTagCache struct {
	lookups map[string]*CacheEntry
}

func NewSearchTagCache() *SearchTagCache {
	return &SearchTagCache{
		lookups: map[string]*CacheEntry{},
	}
}

func (s *SearchTagCache) GetNames() []string {
	tags := make([]string, 0, len(s.lookups))
	for k := range s.lookups {
		tags = append(tags, k)
	}
	sort.Strings(tags)
	return tags
}

func (s *SearchTagCache) GetValues(tagName string) []string {
	if e := s.lookups[tagName]; e != nil {
		vals := make([]string, 0, len(e.values))
		for v := range e.values {
			vals = append(vals, v)
		}
		sort.Strings(vals)
		return vals
	}
	return nil
}

func (s *SearchTagCache) SetData(ts time.Time, data *tempofb.SearchData) {
	tsUnix := ts.Unix()
	kv := &tempofb.KeyValues{}

	l := data.TagsLength()
	for j := 0; j < l; j++ {
		data.Tags(kv, j)
		key := string(kv.Key())
		l2 := kv.ValueLength()
		for k := 0; k < l2; k++ {
			s.setEntry(tsUnix, key, string(kv.Value(k)))
		}
	}
}

func (s *SearchTagCache) setEntry(ts int64, k, v string) {
	e := s.lookups[k]
	if e == nil {
		// First entry
		s.lookups[k] = &CacheEntry{values: map[string]int64{v: ts}}
		return
	}

	// Prune oldest as needed
	for len(e.values) >= maxValuesPerTag {
		earliestv := ""
		earliestts := int64(math.MaxInt64)

		for v, ts := range e.values {
			if ts < earliestts {
				earliestv = v
				earliestts = ts
			}
		}

		delete(e.values, earliestv)
	}

	e.values[v] = ts
}

func (s *SearchTagCache) PurgeExpired(before time.Time) {
	beforeUnix := before.Unix()

	for k, e := range s.lookups {
		for v, ts := range e.values {
			if ts < beforeUnix {
				delete(e.values, v)
			}
		}

		// Remove tags when all values deleted
		if len(e.values) <= 0 {
			delete(s.lookups, k)
		}
	}
}
