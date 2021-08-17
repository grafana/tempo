package search

import (
	"bytes"
	"context"
	"io"

	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
)

var _ SearchBlock = (*BackendSearchBlock)(nil)

type BackendSearchBlock struct {
	id       uuid.UUID
	tenantID string
	l        *local.Backend
}

// NewBackendSearchBlock iterates through the given WAL search data and writes it to the persistent backend
// in a more efficient paged form. Multiple traces are written in the same page to make sure of the flatbuffer
// CreateSharedString feature which dedupes strings across the entire buffer.
func NewBackendSearchBlock(input *StreamingSearchBlock, l *local.Backend, blockID uuid.UUID, tenantID string) error {
	var err error
	ctx := context.TODO()
	indexPageSize := 100 * 1024
	kv := &tempofb.KeyValues{} // buffer

	// Copy records into the appender
	a := encoding.NewBufferedAppenderGeneric(NewBackendSearchBbackendSearchBlockWriter(blockID, tenantID, l), 1024*1024)
	for _, r := range input.appender.Records() {

		// Read
		buf := make([]byte, r.Length)
		_, err = input.file.ReadAt(buf, int64(r.Start))
		if err != nil {
			return err
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

		err = a.Append(ctx, r.ID, data)
		if err != nil {
			return err
		}
	}

	err = a.Complete(ctx)
	if err != nil {
		return err
	}

	// Write index
	ir := a.Records()
	i := encoding.LatestEncoding().NewIndexWriter(indexPageSize)
	indexBytes, err := i.Write(ir)
	if err != nil {
		return err
	}
	err = l.Write(ctx, "search-index", backend.KeyPathForBlock(blockID, tenantID), bytes.NewReader(indexBytes), int64(len(indexBytes)), true)
	if err != nil {
		return err
	}

	// Write meta
	sm := &SearchBlockMeta{
		IndexPageSize: uint32(indexPageSize),
		IndexRecords:  uint32(len(ir)),
		Version:       encoding.LatestEncoding().Version(),
	}
	return WriteSearchBlockMeta(ctx, l, blockID, tenantID, sm)
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
	var pageBuf []byte
	var dataBuf []byte
	entry := &tempofb.SearchData{} // Buffer
	keyPath := backend.KeyPathForBlock(s.id, s.tenantID)

	meta, err := ReadSearchBlockMeta(ctx, s.l, s.id, s.tenantID)
	if err != nil {
		return err
	}

	enc, err := encoding.FromVersion(meta.Version)
	if err != nil {
		return err
	}

	// Read index
	bmeta := backend.NewBlockMeta(s.tenantID, s.id, meta.Version, backend.EncNone, "")
	cr := backend.NewContextReader(bmeta, "search-index", backend.NewReader(s.l), false)

	ir, err := enc.NewIndexReader(cr, int(meta.IndexPageSize), int(meta.IndexRecords))
	if err != nil {
		return err
	}

	i := -1

	for !sr.Quit() {

		i++

		// Next index entry
		record, _ := ir.At(ctx, i)
		if record == nil {
			return nil
		}

		// Reset/resize buffer
		size := record.Length
		if cap(pageBuf) < int(size) {
			pageBuf = make([]byte, size)
		}
		pageBuf = pageBuf[:size]

		// Read page
		err = s.l.ReadRange(ctx, "search", keyPath, record.Start, pageBuf)
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

		sr.AddBytesInspected(uint64(len(dataBuf)))

		datas := tempofb.GetRootAsBatchSearchData(dataBuf, 0)
		l := datas.EntriesLength()
		for j := 0; j < l; j++ {
			sr.AddTraceInspected()

			datas.Entries(entry, j)

			if !p.Matches(entry) {
				continue
			}

			// If we got here then it's a match.
			match := GetSearchResultFromData(entry)

			if quit := sr.AddResult(ctx, match); quit {
				return nil
			}
		}
	}

	return nil
}
