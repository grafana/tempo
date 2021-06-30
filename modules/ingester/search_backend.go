package ingester

import (
	"context"
	"io"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	tempofb "github.com/grafana/tempo/pkg/tempofb/Tempo"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

//var _ SearchBlock = (*searchDataBackend)(nil)

/*type searchWriterBuffered struct {
	tracker backend.AppendTracker
	a       encoding.Appender
	Builder *flatbuffers.Builder
	page    flatbuffers.UOffsetT
	entries []flatbuffers.UOffsetT
}

var _ common.DataWriter = (*searchWriterBuffered)(nil)

func NewSearchWriterBuffered() *searchWriterBuffered {
	return &searchWriterBuffered{
		Builder: flatbuffers.NewBuilder(1024),
	}
}

/*func (*searchWriterBuffered) Complete() error {
	return nil
}

func (s *searchWriterBuffered) CutPage() (int, error) {

	s.builder.Finish(s.page)
	buf := s.builder.FinishedBytes()

	s.

		// Reset for next page
		s.builder.Reset()
	s.entries = s.entries[:0]

	return b, nil
}

func (s *searchWriterBuffered) Write(id common.ID, obj []byte) (int, error) {
	var err error

	return len(obj), err
}
*/

func CompleteSearchDataForBlock(input *searchData, w backend.Writer, block *encoding.BackendBlock) error {
	var err error
	var pageEntries []flatbuffers.UOffsetT
	var tracker backend.AppendTracker

	ctx := context.TODO()
	builder := flatbuffers.NewBuilder(1024)
	pageSize := 1024 * 1024 //1MB

	flush := func() error {
		// Create vector of entries
		tempofb.BatchSearchDataStartEntriesVector(builder, len(pageEntries))
		for _, entry := range pageEntries {
			builder.PrependUOffsetT(entry)
		}
		entryVector := builder.EndVector(len(pageEntries))

		// Write final batch
		tempofb.BatchSearchDataStart(builder)
		tempofb.BatchSearchDataAddEntries(builder, entryVector)
		batch := tempofb.BatchSearchDataEnd(builder)
		builder.FinishSizePrefixed(batch)

		buf := builder.FinishedBytes()

		tracker, err = w.Append(ctx, "search", block.BlockMeta().BlockID, block.BlockMeta().TenantID, tracker, buf)
		if err != nil {
			return err
		}

		// Reset for next page
		builder.Reset()
		pageEntries = pageEntries[:0]

		return nil
	}

	for _, r := range input.appender.Records() {

		// Read data and copy into the new builder
		buf := make([]byte, r.Length)
		_, err = input.file.ReadAt(buf, int64(r.Start))
		if err != nil {
			return err
		}

		tags := tempofb.SearchDataMap{}
		kv := &tempofb.KeyValues{}

		s := tempofb.SearchDataFromBytes(buf)
		for i := 0; i < s.TagsLength(); i++ {
			s.Tags(kv, i)
			for j := 0; j < kv.ValueLength(); j++ {
				tempofb.SearchDataAppend(tags, string(kv.Key()), string(kv.Value(j)))
			}
		}

		offset := tempofb.WriteSearchDataToBuilder(builder, r.ID, tags, s.StartTimeUnixNano(), s.EndTimeUnixNano())
		pageEntries = append(pageEntries, offset)

		if builder.Offset() > flatbuffers.UOffsetT(pageSize) {
			flush()
		}
	}

	// Final page
	if len(pageEntries) > 0 {
		flush()
	}

	err = w.CloseAppend(ctx, tracker)
	if err != nil {
		return err
	}

	return nil
}

type searchDataBackend struct {
	id       uuid.UUID
	tenantID string
	r        backend.Reader
}

func SearchDataFromBlock(r backend.Reader, b *encoding.BackendBlock) *searchDataBackend {
	return &searchDataBackend{
		id:       b.BlockMeta().BlockID,
		tenantID: b.BlockMeta().TenantID,
		r:        r,
	}
}

func (s *searchDataBackend) Search(ctx context.Context, p pipeline) ([]*tempopb.TraceSearchMetadata, error) {

	var matches []*tempopb.TraceSearchMetadata

	offset := uint64(0)

	for {

		// Read page size
		offsetBuf := make([]byte, 4)
		err := s.r.ReadRange(ctx, "search", s.id, s.tenantID, offset, offsetBuf)
		if err == io.EOF {
			return matches, nil
		}
		if err != nil {
			return nil, err
		}

		offset += 4

		size := flatbuffers.GetSizePrefix(offsetBuf, 0)

		// Read page
		dataBuf := make([]byte, size)
		err = s.r.ReadRange(ctx, "search", s.id, s.tenantID, offset, dataBuf)
		if err == io.EOF {
			return matches, nil
		}
		if err != nil {
			return nil, err
		}

		datas := tempofb.GetRootAsBatchSearchData(dataBuf, 0)
		entry := &tempofb.SearchData{}
		for i := 0; i < datas.EntriesLength(); i++ {
			datas.Entries(entry, i)

			if !p.Matches(entry) {
				continue
			}

			// If we got here then it's a match.
			matches = append(matches, &tempopb.TraceSearchMetadata{
				TraceID:           entry.Id(),
				RootServiceName:   tempofb.SearchDataGet(entry, "root.service.name"),
				RootTraceName:     tempofb.SearchDataGet(entry, "root.name"),
				StartTimeUnixNano: entry.StartTimeUnixNano(),
				DurationMs:        uint32((entry.EndTimeUnixNano() - entry.StartTimeUnixNano()) / 1_000_000),
			})

			if len(matches) >= 20 {
				return matches, nil
			}
		}

		offset += uint64(size)
	}
}
