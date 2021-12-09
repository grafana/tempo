package tempofb

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

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
		s.Tags = NewSearchDataMap()
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
	offset := s.WriteToBuilder(b, nil)
	b.Finish(offset)
	return b.FinishedBytes()
}

func (s *SearchEntryMutable) WriteToBuilder(b *flatbuffers.Builder, kvCache map[uint64]flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if s.Tags == nil {
		s.Tags = NewSearchDataMap()
	}

	idOffset := b.CreateByteString(s.TraceID)

	tagOffset := WriteSearchDataMap(b, s.Tags, kvCache)

	SearchEntryStart(b)
	SearchEntryAddId(b, idOffset)
	SearchEntryAddStartTimeUnixNano(b, s.StartTimeUnixNano)
	SearchEntryAddEndTimeUnixNano(b, s.EndTimeUnixNano)
	SearchEntryAddTags(b, tagOffset)
	return SearchEntryEnd(b)
}
