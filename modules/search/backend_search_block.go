package search

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/golang/snappy"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

var _ SearchBlock = (*BackendSearchBlock)(nil)

type BackendSearchBlock struct {
	id       uuid.UUID
	tenantID string
	l        *local.Backend
}

func snappyEncode(dst []byte, src []byte) []byte {
	dst = dst[:0]
	return snappy.Encode(dst, src)
}
func snappyDecode(dst []byte, src []byte) ([]byte, error) {
	dst = dst[:0]
	return snappy.Decode(dst, src)
}

func noEncode(dst []byte, src []byte) []byte {
	return src
}
func noDecode(dst []byte, src []byte) ([]byte, error) {
	return src, nil
}

//var defaultEncode func([]byte, []byte) []byte = snappyEncode
//var defaultDecode func(dst []byte, src []byte) ([]byte, error) = snappyDecode

var defaultEncode func([]byte, []byte) []byte = noEncode
var defaultDecode func(dst []byte, src []byte) ([]byte, error) = noDecode

// NewBackendSearchBlock iterates through the given WAL search data and writes it to the persistent backend
// in a more efficient paged form. Multiple traces are written in the same page to make sure of the flatbuffer
// CreateSharedString feature which dedupes strings across the entire buffer.
// TODO - Use the existing buffered encoder for this?  May need to be refactored, because it currently
//        takes bytes, but we need to pass the search data before bytes...?
func NewBackendSearchBlock(input *StreamingSearchBlock, l *local.Backend, blockID uuid.UUID, tenantID string) (int, error) {
	var err error
	var pageEntries []flatbuffers.UOffsetT
	var tracker backend.AppendTracker

	ctx := context.TODO()
	builder := flatbuffers.NewBuilder(1024)
	pageSize := 1024 * 1024 //1MB
	sizeBuf := make([]byte, 4)
	bytesFlushed := 0
	kv := &tempofb.KeyValues{}
	pageBuf := make([]byte, 0, 1024*1024)

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
		builder.Finish(batch)

		buf := builder.FinishedBytes()

		pageBuf = defaultEncode(pageBuf, buf)

		binary.LittleEndian.PutUint32(sizeBuf, uint32(len(pageBuf)))
		tracker, err = l.Append(ctx, "search", backend.KeyPathForBlock(blockID, tenantID), tracker, sizeBuf)
		if err != nil {
			return err
		}

		tracker, err = l.Append(ctx, "search", backend.KeyPathForBlock(blockID, tenantID), tracker, pageBuf)
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

		s := tempofb.SearchDataFromBytes(buf)

		data := &tempofb.SearchDataMutable{
			TraceID:           r.ID,
			StartTimeUnixNano: s.StartTimeUnixNano(),
			EndTimeUnixNano:   s.EndTimeUnixNano(),
		}

		l := s.TagsLength()
		for i := 0; i < l; i++ {
			s.Tags(kv, i)
			for j := 0; j < kv.ValueLength(); j++ {
				data.AddTag(string(kv.Key()), string(kv.Value(j)))
			}
		}

		offset := data.WriteToBuilder(builder)
		pageEntries = append(pageEntries, offset)

		if builder.Offset() > flatbuffers.UOffsetT(pageSize) {
			err = flush()
			if err != nil {
				return bytesFlushed, err
			}
		}
	}

	// Final page
	if len(pageEntries) > 0 {
		err = flush()
		if err != nil {
			return bytesFlushed, err
		}
	}

	err = l.CloseAppend(ctx, tracker)
	if err != nil {
		return bytesFlushed, err
	}

	return bytesFlushed, nil
}

// OpenBackendSearchBlock opens the search data for an existing block in the given backend.
func OpenBackendSearchBlock(l *local.Backend, blockID uuid.UUID, tenantID string) *BackendSearchBlock {
	return &BackendSearchBlock{
		id:       blockID,
		tenantID: tenantID,
		l:        l,
	}
}

// Search iterates through the block looking for matches.
func (s *BackendSearchBlock) Search(ctx context.Context, p Pipeline, sr *SearchResults) error {

	offset := uint64(0)
	offsetBuf := make([]byte, 4)
	pageBuf := make([]byte, 1024*1024)
	dataBuf := make([]byte, 1024*1024)
	entry := &tempofb.SearchData{} // Buffer

	for !sr.Quit() {

		// Read page size
		err := s.l.ReadRange(ctx, "search", backend.KeyPathForBlock(s.id, s.tenantID), offset, offsetBuf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		offset += 4

		size := binary.LittleEndian.Uint32(offsetBuf)

		// Reset/resize buffer
		if cap(pageBuf) < int(size) {
			pageBuf = make([]byte, size)
		}
		pageBuf = pageBuf[:size]

		// Read page
		err = s.l.ReadRange(ctx, "search", backend.KeyPathForBlock(s.id, s.tenantID), offset, pageBuf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		dataBuf, err = defaultDecode(dataBuf, pageBuf)
		if err != nil {
			return err
		}

		sr.AddBytesInspected(uint64(size))

		datas := tempofb.GetRootAsBatchSearchData(dataBuf, 0)
		l := datas.EntriesLength()
		for i := 0; i < l; i++ {
			sr.AddTraceInspected()

			datas.Entries(entry, i)

			if !p.Matches(entry) {
				continue
			}

			// If we got here then it's a match.
			match := GetSearchResultFromData(entry)

			if quit := sr.AddResult(ctx, match); quit {
				return nil
			}
		}

		offset += uint64(size)
	}

	return nil
}
