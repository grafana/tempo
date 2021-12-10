package tempofb

import (
	"hash"
	"sort"
	"strings"

	"github.com/cespare/xxhash"
	flatbuffers "github.com/google/flatbuffers/go"
)

type SearchDataMap map[string]map[string]struct{}

func NewSearchDataMap() SearchDataMap {
	return make(SearchDataMap, 10) // 10 for luck
}

func NewSearchDataMapWithData(m map[string][]string) SearchDataMap {
	s := NewSearchDataMap()

	for k, vv := range m {
		for _, v := range vv {
			s.Add(k, v)
		}
	}
	return s
}

func (s SearchDataMap) Add(k, v string) {
	values, ok := s[k]
	if !ok {
		// first entry
		s[k] = map[string]struct{}{v: {}}
		return
	}

	// For repeats it is more performant to avoid the map assigns.
	if _, ok = values[v]; !ok {
		values[v] = struct{}{}
	}
}

func (s SearchDataMap) Contains(k, v string) bool {
	if values, ok := s[k]; ok {
		_, ok := values[v]
		return ok
	}
	return false
}

func (s SearchDataMap) Range(f func(k, v string)) {
	for k, values := range s {
		for v := range values {
			f(k, v)
		}
	}
}

func (s SearchDataMap) RangeKeys(f func(k string)) {
	for k := range s {
		f(k)
	}
}

func (s SearchDataMap) RangeKeyValues(k string, f func(v string)) {
	for v := range s[k] {
		f(v)
	}
}

func WriteSearchDataMap(b *flatbuffers.Builder, d SearchDataMap, cache map[uint64]flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	h := xxhash.New()

	var keys []string
	d.RangeKeys(func(k string) {
		keys = append(keys, k)
	})
	sort.Strings(keys)

	offsets := make([]flatbuffers.UOffsetT, 0, len(keys))
	var values []string
	for _, k := range keys {

		values = values[:0]
		d.RangeKeyValues(k, func(v string) {
			values = append(values, v)
		})

		offsets = append(offsets, writeKeyValues(b, k, values, h, cache))
	}

	SearchEntryStartTagsVector(b, len(offsets))
	for _, kvo := range offsets {
		b.PrependUOffsetT(kvo)
	}
	vector := b.EndVector((len(offsets)))
	return vector
}

// writeKeyValues saves the key->values entry to the builder.  Results are optionally cached and
// existing identical key->values entries reused.
func writeKeyValues(b *flatbuffers.Builder, key string, values []string, h hash.Hash64, cache map[uint64]flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	// Skip empty keys
	if len(values) <= 0 {
		return 0
	}

	// Preparation, must be done before hashing/caching.
	key = strings.ToLower(key)
	for i := range values {
		values[i] = strings.ToLower(values[i])
	}
	sort.Strings(values)

	// Hash, cache (optional)
	var ce uint64
	if cache != nil {
		h.Reset()
		h.Write([]byte(key))
		for _, v := range values {
			h.Write([]byte{0}) // separator
			h.Write([]byte(v))
		}
		ce = h.Sum64()
		if offset, ok := cache[ce]; ok {
			return offset
		}
	}

	ko := b.CreateSharedString(key)
	valueStrings := make([]flatbuffers.UOffsetT, len(values))
	for i := range values {
		valueStrings[i] = b.CreateSharedString(strings.ToLower(values[i]))
	}

	KeyValuesStartValueVector(b, len(valueStrings))
	for _, vs := range valueStrings {
		b.PrependUOffsetT(vs)
	}
	valueVector := b.EndVector(len(valueStrings))

	KeyValuesStart(b)
	KeyValuesAddKey(b, ko)
	KeyValuesAddValue(b, valueVector)
	offset := KeyValuesEnd(b)

	if cache != nil {
		cache[ce] = offset
	}

	return offset
}
