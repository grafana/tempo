package tempofb

import (
	"bytes"
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

func SearchDataContains(s *SearchData, k string, v string) bool {
	kv := &KeyValues{}
	kb := bytes.ToLower([]byte(k))
	vb := bytes.ToLower([]byte(v))

	// TODO - Use binary search since keys/values are sorted
	for i := 0; i < s.TagsLength(); i++ {
		s.Tags(kv, i)
		if bytes.Equal(kv.Key(), kb) {
			for j := 0; j < kv.ValueLength(); j++ {
				if bytes.Contains(kv.Value(j), vb) {
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

	for k, v := range tags {
		ko := b.CreateSharedString(strings.ToLower(k))

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
