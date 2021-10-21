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
	//return &SearchDataMap1{}
	return &SearchDataMap2{}
}

type SearchDataMap1 map[string][]string

func (s SearchDataMap1) Add(k, v string) {
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

func (s SearchDataMap1) Contains(k, v string) bool {
	e := s[k]
	if e != nil {
		vv := string(v)
		for _, vvv := range e {
			if strings.Contains(vvv, vv) {
				return true
			}
		}
	}

	return false
}

func (s SearchDataMap1) Range(f func(k, v string)) {
	for k, vv := range s {
		for _, v := range vv {
			f(k, v)
		}
	}
}

func (s SearchDataMap1) WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT {
	offsets := make([]flatbuffers.UOffsetT, 0, len(s))

	// Sort keys
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// Skip empty keys
		if len(s[k]) <= 0 {
			continue
		}

		ko := b.CreateSharedString(strings.ToLower(k))

		// Sort values
		v := s[k]
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
		offsets = append(offsets, KeyValuesEnd(b))
	}

	SearchEntryStartTagsVector(b, len(offsets))
	for _, kvo := range offsets {
		b.PrependUOffsetT(kvo)
	}
	keyValueVector := b.EndVector((len(offsets)))
	return keyValueVector
}

type SearchDataMap2 map[string]map[string]struct{}

func (s SearchDataMap2) Add(k, v string) {
	values, ok := s[k]
	if !ok {
		// first entry
		m := make(map[string]struct{}, 10)
		m[v] = struct{}{}
		s[k] = m
		//s[k] = map[string]struct{}{v: {}}
		return
	}

	if _, ok = values[v]; !ok {
		values[v] = struct{}{}
	}
	//kk[v] = struct{}{}
}

func (s SearchDataMap2) Contains(k, v string) bool {
	if values, ok := s[k]; ok {
		_, ok := values[v]
		return ok
	}
	return false
}

func (s SearchDataMap2) Range(f func(k, v string)) {
	for k, values := range s {
		for v := range values {
			f(k, v)
		}
	}
}

func (s SearchDataMap2) WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT {
	offsets := make([]flatbuffers.UOffsetT, 0, len(s))

	// Sort keys
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// Skip empty keys
		if len(s[k]) <= 0 {
			continue
		}

		ko := b.CreateSharedString(strings.ToLower(k))

		// Sort values
		values := s[k]
		v := make([]string, 0, len(values))
		for vv := range values {
			v = append(v, vv)
		}
		//v := s[k]
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
		offsets = append(offsets, KeyValuesEnd(b))
	}

	SearchEntryStartTagsVector(b, len(offsets))
	for _, kvo := range offsets {
		b.PrependUOffsetT(kvo)
	}
	keyValueVector := b.EndVector((len(offsets)))
	return keyValueVector
}
