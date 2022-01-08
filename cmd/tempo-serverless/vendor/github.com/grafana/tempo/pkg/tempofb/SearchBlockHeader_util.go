package tempofb

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

func (s *SearchBlockHeader) Contains(k []byte, v []byte, buffer *KeyValues) bool {
	return ContainsTag(s, buffer, k, v)
}

type SearchBlockHeaderMutable struct {
	Tags   SearchDataMap
	MinDur uint64
	MaxDur uint64
}

func NewSearchBlockHeaderMutable() *SearchBlockHeaderMutable {
	return &SearchBlockHeaderMutable{
		Tags: NewSearchDataMap(),
	}
}

func (s *SearchBlockHeaderMutable) AddEntry(e *SearchEntry) {

	kv := &KeyValues{} //buffer

	// Record all unique keyvalues
	for i, ii := 0, e.TagsLength(); i < ii; i++ {
		e.Tags(kv, i)
		key := string(kv.Key())
		for j, jj := 0, kv.ValueLength(); j < jj; j++ {
			s.AddTag(key, string(kv.Value(j)))
		}
	}

	// Record min/max durations
	dur := e.EndTimeUnixNano() - e.StartTimeUnixNano()
	if s.MinDur == 0 || dur < s.MinDur {
		s.MinDur = dur
	}
	if dur > s.MaxDur {
		s.MaxDur = dur
	}
}

// AddTag adds the unique tag name and value to the search data. No effect if the pair is already present.
func (s *SearchBlockHeaderMutable) AddTag(k string, v string) {
	s.Tags.Add(k, v)
}

func (s *SearchBlockHeaderMutable) MinDurationNanos() uint64 {
	return s.MinDur
}

func (s *SearchBlockHeaderMutable) MaxDurationNanos() uint64 {
	return s.MaxDur
}

func (s *SearchBlockHeaderMutable) Contains(k []byte, v []byte, _ *KeyValues) bool {
	return s.Tags.Contains(string(k), string(v))
}

func (s *SearchBlockHeaderMutable) ToBytes() []byte {
	b := flatbuffers.NewBuilder(1024)

	tags := WriteSearchDataMap(b, s.Tags, nil)

	SearchBlockHeaderStart(b)
	SearchBlockHeaderAddMinDurationNanos(b, s.MinDur)
	SearchBlockHeaderAddMaxDurationNanos(b, s.MaxDur)
	SearchBlockHeaderAddTags(b, tags)
	offset := SearchBlockHeaderEnd(b)
	b.Finish(offset)
	return b.FinishedBytes()
}
