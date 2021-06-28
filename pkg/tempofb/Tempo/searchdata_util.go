package Tempo

import (
	"bytes"
	"strings"

	flatbuffers "github.com/google/flatbuffers/go"
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

func SearchDataContains(s *SearchData, k string, v string) bool {
	kv := &KeyValues{}
	kb := bytes.ToLower([]byte(k))
	vb := bytes.ToLower([]byte(v))

	for i := 0; i < s.DataLength(); i++ {
		s.Data(kv, i)
		if bytes.Compare(kv.Key(), kb) == 0 {
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

func SearchDataFromMap(d SearchDataMap) []byte {
	b := flatbuffers.NewBuilder(2048)

	keyValueOffsets := make([]flatbuffers.UOffsetT, 0, len(d))

	for k, v := range d {
		ko := b.CreateString(strings.ToLower(k))

		valueStrings := make([]flatbuffers.UOffsetT, len(v))
		for i := range v {
			valueStrings[i] = b.CreateString(strings.ToLower(v[i]))
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

	SearchDataStartDataVector(b, len(keyValueOffsets))
	for _, kvo := range keyValueOffsets {
		b.PrependUOffsetT(kvo)
	}
	keyValueVector := b.EndVector((len(keyValueOffsets)))

	SearchDataStart(b)
	SearchDataAddData(b, keyValueVector)
	s := SearchDataEnd(b)
	b.Finish(s)
	return b.FinishedBytes()
}
