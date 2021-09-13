package tempofb

import flatbuffers "github.com/google/flatbuffers/go"

type SearchBlockHeaderBuilder struct {
	Tags   SearchDataMap
	MinDur uint64
	MaxDur uint64
}

func NewSearchBlockHeaderBuilder() *SearchBlockHeaderBuilder {
	return &SearchBlockHeaderBuilder{
		Tags: SearchDataMap{},
	}
}

func (s *SearchBlockHeaderBuilder) AddEntry(e *SearchEntry) {

	kv := &KeyValues{} //buffer

	// Record all unique keyvalues
	for i, ii := 0, e.TagsLength(); i < ii; i++ {
		e.Tags(kv, i)
		for j, jj := 0, kv.ValueLength(); j < jj; j++ {
			s.AddTag(string(kv.Key()), string(kv.Value(j)))
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
func (s *SearchBlockHeaderBuilder) AddTag(k string, v string) {
	s.Tags.Add(k, v)
}

func (s *SearchBlockHeaderBuilder) ToBytes() []byte {
	b := flatbuffers.NewBuilder(1024)

	tags := s.Tags.WriteToBuilder(b)

	SearchBlockHeaderStart(b)
	SearchBlockHeaderAddMinDurationNanos(b, s.MinDur)
	SearchBlockHeaderAddMaxDurationNanos(b, s.MaxDur)
	SearchBlockHeaderAddTags(b, tags)
	offset := SearchBlockHeaderEnd(b)
	b.Finish(offset)
	return b.FinishedBytes()
}
