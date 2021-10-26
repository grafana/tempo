package tempofb

import (
	"sort"
	"strings"

	flatbuffers "github.com/google/flatbuffers/go"
)

type SearchDataMap interface {
	Add(k, v string)
	Contains(k, v string) bool
	WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT
	Range(f func(k, v string))
}

func NewSearchDataMap() SearchDataMap {
	return make(SearchDataMapLarge, 10) // 10 for luck
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

type SearchDataMapSmall map[string][]string

func (s SearchDataMapSmall) Add(k, v string) {
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

func (s SearchDataMapSmall) Contains(k, v string) bool {
	e := s[k]
	for _, vvv := range e {
		if strings.Contains(vvv, v) {
			return true
		}
	}

	return false
}

func (s SearchDataMapSmall) Range(f func(k, v string)) {
	for k, vv := range s {
		for _, v := range vv {
			f(k, v)
		}
	}
}

func (s SearchDataMapSmall) WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	valuesf := func(k string, _ []string) []string {
		return s[k]
	}

	return writeToBuilder(b, keys, valuesf)
}

type SearchDataMapLarge map[string]map[string]struct{}

func (s SearchDataMapLarge) Add(k, v string) {
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

func (s SearchDataMapLarge) Contains(k, v string) bool {
	if values, ok := s[k]; ok {
		_, ok := values[v]
		return ok
	}
	return false
}

func (s SearchDataMapLarge) Range(f func(k, v string)) {
	for k, values := range s {
		for v := range values {
			f(k, v)
		}
	}
}

func (s SearchDataMapLarge) WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	valuesf := func(k string, buffer []string) []string {
		buffer = buffer[:0]
		for v := range s[k] {
			buffer = append(buffer, v)
		}

		return buffer
	}

	return writeToBuilder(b, keys, valuesf)
}

func writeToBuilder(b *flatbuffers.Builder, keys []string, valuesf func(k string, buffer []string) []string) flatbuffers.UOffsetT {

	var values []string

	offsets := make([]flatbuffers.UOffsetT, 0, len(keys))
	sort.Strings(keys)

	for _, k := range keys {

		values := valuesf(k, values)

		// Skip empty keys
		if len(values) <= 0 {
			continue
		}

		ko := b.CreateSharedString(strings.ToLower(k))

		// Sort values
		sort.Strings(values)

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
		offsets = append(offsets, KeyValuesEnd(b))
	}

	SearchEntryStartTagsVector(b, len(offsets))
	for _, kvo := range offsets {
		b.PrependUOffsetT(kvo)
	}
	keyValueVector := b.EndVector((len(offsets)))
	return keyValueVector
}
