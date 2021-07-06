package search

import (
	"context"
	"io"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
)

var _ SearchBlock = (*BackendSearchBlock)(nil)

type BackendSearchBlock struct {
	id       uuid.UUID
	tenantID string
	r        backend.Reader
}

// NewBackendSearchBlock iterates through the given WAL search data and writes it to the persistent backend
// in a more efficient paged form. Multiple traces are written in the same page to make sure of the flatbuffer
// CreateSharedString feature which dedupes strings across the entire buffer.
// TODO - Use the existing buffered encoder for this?  May need to be refactored, because it currently
//        takes bytes, but we need to pass the search data before bytes...?
func NewBackendSearchBlock(input *StreamingSearchBlock, w backend.Writer, blockID uuid.UUID, tenantID string) (int, error) {
	var err error
	var pageEntries []flatbuffers.UOffsetT
	var tracker backend.AppendTracker

	ctx := context.TODO()
	builder := flatbuffers.NewBuilder(1024)
	pageSize := 1024 * 1024 //1MB
	bytesFlushed := 0

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

		tracker, err = w.Append(ctx, "search", blockID, tenantID, tracker, buf)
		if err != nil {
			return err
		}

		bytesFlushed += len(buf)

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
			return bytesFlushed, err
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
		return bytesFlushed, err
	}

	return bytesFlushed, nil
}

// OpenBackendSearchBlock opens the search data for an existing block in the given backend.
func OpenBackendSearchBlock(r backend.Reader, blockID uuid.UUID, tenantID string) *BackendSearchBlock {
	return &BackendSearchBlock{
		id:       blockID,
		tenantID: tenantID,
		r:        r,
	}
}

// Search iterates through the block looking for matches.
func (s *BackendSearchBlock) Search(ctx context.Context, p Pipeline) ([]*tempopb.TraceSearchMetadata, error) {

	var matches []*tempopb.TraceSearchMetadata

	offset := uint64(0)
	offsetBuf := make([]byte, 4)
	dataBuf := make([]byte, 1024*1024)
	entry := &tempofb.SearchData{} // Buffer

	for {

		// Read page size
		err := s.r.ReadRange(ctx, "search", s.id, s.tenantID, offset, offsetBuf)
		if err == io.EOF {
			return matches, nil
		}
		if err != nil {
			return nil, err
		}

		offset += 4

		size := flatbuffers.GetSizePrefix(offsetBuf, 0)

		// Reset/resize buffer
		if cap(dataBuf) < int(size) {
			dataBuf = make([]byte, size)
		}
		dataBuf = dataBuf[:size]

		//fmt.Println("BackendSearchBlock is loading a page size", size, "bytes")

		// Read page
		err = s.r.ReadRange(ctx, "search", s.id, s.tenantID, offset, dataBuf)
		if err == io.EOF {
			return matches, nil
		}
		if err != nil {
			return nil, err
		}

		datas := tempofb.GetRootAsBatchSearchData(dataBuf, 0)
		l := datas.EntriesLength()
		for i := 0; i < l; i++ {
			datas.Entries(entry, i)

			if !p.Matches(entry) {
				continue
			}

			// If we got here then it's a match.
			matches = append(matches, &tempopb.TraceSearchMetadata{
				TraceID:           util.TraceIDToHexString(entry.Id()),
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
