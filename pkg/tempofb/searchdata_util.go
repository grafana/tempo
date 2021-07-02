package tempofb

import (
	"bytes"
	"sort"
	"strings"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type SearchDataMap map[string][]string

func SearchDataAppend(d SearchDataMap, k string, v string) {
	vs, ok := d[k]
	if !ok {
		// First entry for key
		d[k] = []string{v}
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
	d[k] = append(vs, v)
}

// SearchDataGet searches SearchData and returns the first value found for the given key.
func SearchDataGet(s *SearchData, k string) string {
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
func SearchDataContains(s *SearchData, kv *KeyValues, k []byte, v []byte) bool {

	// Binary search for keys. Flatbuffers are written backwards so
	// keys are descending.
	keyIndex := sort.Search(s.TagsLength(), func(i int) bool {
		s.Tags(kv, i)
		return bytes.Compare(k, kv.Key()) >= 0
	})

	if keyIndex < s.TagsLength() && keyIndex >= 0 {
		s.Tags(kv, keyIndex)
		if bytes.Equal(kv.Key(), k) {
			// Linear search for matching values
			for j := 0; j < kv.ValueLength(); j++ {
				if bytes.Contains(kv.Value(j), v) {
					return true
				}
			}
		}
	}

	return false
}

func SearchDataFromBytes(b []byte) *SearchData {
	return GetRootAsSearchData(b, 0)
}

func WriteSearchDataToBuilder(b *flatbuffers.Builder, id common.ID, tags SearchDataMap, startTimeUnixNano, endTimeUnixNano uint64) flatbuffers.UOffsetT {
	keyValueOffsets := make([]flatbuffers.UOffsetT, 0, len(tags))

	idOffset := b.CreateByteString(id)

	// Sort keys
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		ko := b.CreateSharedString(strings.ToLower(k))

		// Sort values
		v := tags[k]
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
	SearchDataAddStartTimeUnixNano(b, startTimeUnixNano)
	SearchDataAddEndTimeUnixNano(b, endTimeUnixNano)
	SearchDataAddTags(b, keyValueVector)
	return SearchDataEnd(b)
}

func SearchDataBytesFromValues(id common.ID, tags SearchDataMap, startTimeUnixNano, endTimeUnixNano uint64) []byte {
	b := flatbuffers.NewBuilder(2048)

	s := WriteSearchDataToBuilder(b, id, tags, startTimeUnixNano, endTimeUnixNano)

	b.Finish(s)

	return b.FinishedBytes()

}
