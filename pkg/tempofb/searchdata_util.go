package tempofb

import (
	"bytes"
	"sort"
	"strings"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type SearchDataMap map[string][]string

func (s SearchDataMap) Add(k, v string) {
	vs, ok := s[k]
	if !ok {
		// First entry for key
		s[k] = []string{v}
		return
	}

	// Key already present, now check for value
	for i := range vs {
		if vs[i] == v {
			// Already present, nothing to do
			return
		}
	}

	// Not found, append
	s[k] = append(vs, v)
}

// SearchDataMutable is a mutable form of the flatbuffer-compiled SearchData struct, to make building and transporting.
type SearchDataMutable struct {
	TraceID           common.ID
	Tags              SearchDataMap
	StartTimeUnixNano uint64
	EndTimeUnixNano   uint64
}

// AddTag adds the unique tag name and value to the search data. No effect if the pair is already present.
func (s *SearchDataMutable) AddTag(k string, v string) {
	if s.Tags == nil {
		s.Tags = SearchDataMap{}
	}
	s.Tags.Add(k, v)
}

// SetStartTimeUnixNano records the earliest of all timestamps passed to this function.
func (s *SearchDataMutable) SetStartTimeUnixNano(t uint64) {
	if t > 0 && s.StartTimeUnixNano == 0 || s.StartTimeUnixNano > t {
		s.StartTimeUnixNano = t
	}
}

// SetEndTimeUnixNano records the latest of all timestamps passed to this function.
func (s *SearchDataMutable) SetEndTimeUnixNano(t uint64) {
	if t > 0 && t > s.EndTimeUnixNano {
		s.EndTimeUnixNano = t
	}
}

func (s *SearchDataMutable) ToBytes() []byte {
	b := flatbuffers.NewBuilder(2048)
	offset := s.WriteToBuilder(b)
	b.Finish(offset)
	return b.FinishedBytes()
}

func (s *SearchDataMutable) WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT {
	keyValueOffsets := make([]flatbuffers.UOffsetT, 0, len(s.Tags))

	idOffset := b.CreateByteString(s.TraceID)

	// Sort keys
	keys := make([]string, 0, len(s.Tags))
	for k := range s.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		ko := b.CreateSharedString(strings.ToLower(k))

		// Sort values
		v := s.Tags[k]
		sort.Strings(v)

		valueStrings := make([]flatbuffers.UOffsetT, len(v))
		for i := range v {
			valueStrings[i] = b.CreateSharedString(strings.ToLower(v[i]))
		}

		KeyValuesStartValueVector(b, len(valueStrings))
		for _, vs := range valueStrings {
			b.PrependUOffsetT(vs)
		}
		valueVector := b.EndVector(len(valueStrings))

		KeyValuesStart(b)
		KeyValuesAddKey(b, ko)
		KeyValuesAddValue(b, valueVector)
		keyValueOffsets = append(keyValueOffsets, KeyValuesEnd(b))
	}

	SearchDataStartTagsVector(b, len(keyValueOffsets))
	for _, kvo := range keyValueOffsets {
		b.PrependUOffsetT(kvo)
	}
	keyValueVector := b.EndVector((len(keyValueOffsets)))

	SearchDataStart(b)
	SearchDataAddId(b, idOffset)
	SearchDataAddStartTimeUnixNano(b, s.StartTimeUnixNano)
	SearchDataAddEndTimeUnixNano(b, s.EndTimeUnixNano)
	SearchDataAddTags(b, keyValueVector)
	return SearchDataEnd(b)
}

// SearchDataGet searches SearchData and returns the first value found for the given key.
func (s *SearchData) Get(k string) string {
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

// SearchDataContains returns true if the key and value are found in the search data.
// Buffer KeyValue object can be passed to reduce allocations. Key and value must be
// already converted to byte slices which match the nature of the flatbuffer data
// which reduces allocations even further.
func (s *SearchData) Contains(kv *KeyValues, k []byte, v []byte) bool {

	matched := -1

	// Binary search for keys. Flatbuffers are written backwards so
	// keys are descending (the comparison is reversed).
	// TODO - We only want exact matches, sort.Search has to make an
	// extra comparison. We should fork it to make use of the full
	// tri-state response from bytes.Compare
	sort.Search(s.TagsLength(), func(i int) bool {
		s.Tags(kv, i)
		comparison := bytes.Compare(k, kv.Key())
		if comparison == 0 {
			matched = i
			// TODO it'd be great to exit here and retain the data in kv buffer
		}
		return comparison >= 0
	})

	if matched >= 0 {
		s.Tags(kv, matched)

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

func SearchDataFromBytes(b []byte) *SearchData {
	return GetRootAsSearchData(b, 0)
}
