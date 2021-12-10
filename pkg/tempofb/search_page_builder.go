package tempofb

import flatbuffers "github.com/google/flatbuffers/go"

type SearchPageBuilder struct {
	builder     *flatbuffers.Builder
	allTags     SearchDataMap
	pageEntries []flatbuffers.UOffsetT
	kvcache     map[uint64]flatbuffers.UOffsetT
}

func NewSearchPageBuilder() *SearchPageBuilder {
	return &SearchPageBuilder{
		builder: flatbuffers.NewBuilder(1024),
		allTags: NewSearchDataMap(),
		kvcache: map[uint64]flatbuffers.UOffsetT{},
	}
}

func (b *SearchPageBuilder) AddData(data *SearchEntryMutable) int {
	if data.Tags != nil {
		data.Tags.Range(func(k, v string) {
			b.allTags.Add(k, v)
		})
	}

	oldOffset := b.builder.Offset()
	offset := data.WriteToBuilder(b.builder, b.kvcache)
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
	tagOffset := WriteSearchDataMap(b.builder, b.allTags, b.kvcache)

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
	b.allTags = NewSearchDataMap()
	b.kvcache = map[uint64]flatbuffers.UOffsetT{}
}
