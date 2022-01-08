package tempofb

import (
	"bytes"

	flatbuffers "github.com/google/flatbuffers/go"
)

// Get searches the entry and returns the first value found for the given key.
func (s *SearchEntry) Get(k string) string {
	kv := &KeyValues{}
	kb := bytes.ToLower([]byte(k))

	// TODO - Use binary search since keys/values are sorted
	for i := 0; i < s.TagsLength(); i++ {
		s.Tags(kv, i)
		if bytes.Equal(kv.Key(), kb) {
			return string(kv.Value(0))
		}
	}

	return ""
}

// Contains returns true if the key and value are found in the search data.
// Buffer KeyValue object can be passed to reduce allocations. Key and value must be
// already converted to byte slices which match the nature of the flatbuffer data
// which reduces allocations even further.
func (s *SearchEntry) Contains(k []byte, v []byte, buffer *KeyValues) bool {
	return ContainsTag(s, buffer, k, v)
}

func (s *SearchEntry) Reset(b []byte) {
	n := flatbuffers.GetUOffsetT(b)
	s.Init(b, n)
}

func NewSearchEntryFromBytes(b []byte) *SearchEntry {
	return GetRootAsSearchEntry(b, 0)
}

type FBTagContainer interface {
	Tags(obj *KeyValues, j int) bool
	TagsLength() int
}

func ContainsTag(s FBTagContainer, kv *KeyValues, k []byte, v []byte) bool {

	kv = FindTag(s, kv, k)
	if kv != nil {
		// Linear search for matching values
		l := kv.ValueLength()
		for j := 0; j < l; j++ {
			if bytes.Contains(kv.Value(j), v) {
				return true
			}
		}
	}

	return false
}

func FindTag(s FBTagContainer, kv *KeyValues, k []byte) *KeyValues {

	idx := binarySearch(s.TagsLength(), func(i int) int {
		s.Tags(kv, i)
		// Note comparison here is backwards because KeyValues are written to flatbuffers in reverse order.
		return bytes.Compare(kv.Key(), k)
	})

	if idx >= 0 {
		// Data is left in buffer when matched
		return kv
	}

	return nil
}

// binarySearch that finds exact matching entry. Returns non-zero index when found, or -1 when not found
// Inspired by sort.Search but makes uses of tri-state comparator to eliminate the last comparison when
// we want to find exact match, not insertion point.
func binarySearch(n int, compare func(int) int) int {
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		switch compare(h) {
		case 0:
			// Found exact match
			return h
		case -1:
			j = h
		case 1:
			i = h + 1
		}
	}

	// No match
	return -1
}
