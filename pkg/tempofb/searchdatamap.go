package tempofb

import (
	"hash"
	"sort"
	"strings"

	"github.com/cespare/xxhash"
	flatbuffers "github.com/google/flatbuffers/go"
)

const (
	// Only save first 1KB of value for searching
	maxValueLen = 1024

	// Soft cap for flatbuffer max buffer size. Signed int32s means the maximum
	// internal size is 2GB, but since the size is checked when growing and doubling,
	// the practical limit is 1GB. 900MB is a soft cap comfortably below which prevents
	// the growth from 1GB->2GB and panic, and also enough room to finish writing
	// outstanding entries and generate a valid (albeit incomplete) struct.
	maxBufferLen = 900 << 20
)

type TagCallback func(t string)

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

	// Max value size
	if len(v) > maxValueLen {
		v = v[0:maxValueLen]
	}

	// For repeats it is more performant to avoid the map assigns.
	if _, ok = values[v]; !ok {
		values[v] = struct{}{}
	}
}

// Contains is an exact match on key, but substring on value
func (s SearchDataMap) Contains(k, v string) bool {
	for vv := range s[k] {
		if strings.Contains(vv, v) {
			return true
		}
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

func (s SearchDataMap) RangeKeys(f TagCallback) {
	for k := range s {
		f(k)
	}
}

func (s SearchDataMap) RangeKeyValues(k string, f TagCallback) {
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

		offset := writeKeyValues(b, k, values, h, cache)
		if offset == 0 {
			continue
		}

		offsets = append(offsets, offset)
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

	// Stop writing new entries when buffer has exceeded
	// the soft cap.
	if b.Offset() > maxBufferLen {
		return 0
	}

	// Preparation, must be done before hashing/caching.
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
		valueStrings[i] = b.CreateSharedString(values[i])
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
