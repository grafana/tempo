package tempofb

import (
	"bytes"
	"sort"
	"strings"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// TagContainer is anything with KeyValues (tags). This is implemented by both
// SearchPage and SearchEntry.
type TagContainer interface {
	Tags(obj *KeyValues, j int) bool
	TagsLength() int
}

var _ TagContainer = (*SearchPage)(nil)
var _ TagContainer = (*SearchEntry)(nil)

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

func (s SearchDataMap) WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT {
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

// SearchEntryMutable is a mutable form of the flatbuffer-compiled SearchEntry struct to make building and transporting easier.
type SearchEntryMutable struct {
	TraceID           common.ID
	Tags              SearchDataMap
	StartTimeUnixNano uint64
	EndTimeUnixNano   uint64
}

// AddTag adds the unique tag name and value to the search data. No effect if the pair is already present.
func (s *SearchEntryMutable) AddTag(k string, v string) {
	if s.Tags == nil {
		s.Tags = SearchDataMap{}
	}
	s.Tags.Add(k, v)
}

// SetStartTimeUnixNano records the earliest of all timestamps passed to this function.
func (s *SearchEntryMutable) SetStartTimeUnixNano(t uint64) {
	if t > 0 && (s.StartTimeUnixNano == 0 || s.StartTimeUnixNano > t) {
		s.StartTimeUnixNano = t
	}
}

// SetEndTimeUnixNano records the latest of all timestamps passed to this function.
func (s *SearchEntryMutable) SetEndTimeUnixNano(t uint64) {
	if t > 0 && t > s.EndTimeUnixNano {
		s.EndTimeUnixNano = t
	}
}

func (s *SearchEntryMutable) ToBytes() []byte {
	b := flatbuffers.NewBuilder(2048)
	offset := s.WriteToBuilder(b)
	b.Finish(offset)
	return b.FinishedBytes()
}

func (s *SearchEntryMutable) WriteToBuilder(b *flatbuffers.Builder) flatbuffers.UOffsetT {

	idOffset := b.CreateByteString(s.TraceID)

	tagOffset := s.Tags.WriteToBuilder(b)

	SearchEntryStart(b)
	SearchEntryAddId(b, idOffset)
	SearchEntryAddStartTimeUnixNano(b, s.StartTimeUnixNano)
	SearchEntryAddEndTimeUnixNano(b, s.EndTimeUnixNano)
	SearchEntryAddTags(b, tagOffset)
	return SearchEntryEnd(b)
}

type SearchPageBuilder struct {
	builder     *flatbuffers.Builder
	allTags     SearchDataMap
	pageEntries []flatbuffers.UOffsetT
}

func NewSearchPageBuilder() *SearchPageBuilder {
	return &SearchPageBuilder{
		builder: flatbuffers.NewBuilder(1024),
		allTags: SearchDataMap{},
	}
}

func (b *SearchPageBuilder) AddData(data *SearchEntryMutable) int {
	for k, vv := range data.Tags {
		for _, v := range vv {
			b.allTags.Add(k, v)
		}
	}

	oldOffset := b.builder.Offset()
	offset := data.WriteToBuilder(b.builder)
	b.pageEntries = append(b.pageEntries, offset)

	// bytes written
	return int(offset - oldOffset)
}

func (b *SearchPageBuilder) Finish() []byte {
	// At this point all individual entries have been written
	// to the fb builder. Now we need to wrap them up in the final
	// batch object.

	// Create vector
	SearchPageStartEntriesVector(b.builder, len(b.pageEntries))
	for _, entry := range b.pageEntries {
		b.builder.PrependUOffsetT(entry)
	}
	entryVector := b.builder.EndVector(len(b.pageEntries))

	// Create batch-level tags
	tagOffset := b.allTags.WriteToBuilder(b.builder)

	// Write final batch object
	SearchPageStart(b.builder)
	SearchPageAddEntries(b.builder, entryVector)
	SearchPageAddTags(b.builder, tagOffset)
	batch := SearchPageEnd(b.builder)
	b.builder.Finish(batch)
	buf := b.builder.FinishedBytes()

	return buf
}

func (b *SearchPageBuilder) Reset() {
	b.builder.Reset()
	b.pageEntries = b.pageEntries[:0]
	b.allTags = SearchDataMap{}
}

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
func (s *SearchEntry) Contains(kv *KeyValues, k []byte, v []byte) bool {

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

func SearchEntryFromBytes(b []byte) *SearchEntry {
	return GetRootAsSearchEntry(b, 0)
}

func ContainsTag(s TagContainer, kv *KeyValues, k []byte, v []byte) bool {

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

func SearchEntryToSearchEntryMutable(entry *SearchEntry) *SearchEntryMutable {
	m := &SearchEntryMutable{
		Tags: SearchDataMap{},
	}

	// TraceID
	m.TraceID = entry.Id()

	// Tags
	kv := &KeyValues{}
	for i := 0; i < entry.TagsLength(); i++ {
		entry.Tags(kv, i)
		for j := 0; j < kv.ValueLength(); j++ {
			m.Tags.Add(string(kv.Key()), string(kv.Value(j)))
		}
	}

	// Start time
	m.StartTimeUnixNano = entry.StartTimeUnixNano()

	// End time
	m.EndTimeUnixNano = entry.EndTimeUnixNano()

	return m
}
