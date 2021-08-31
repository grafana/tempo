package search

import (
	"bytes"
	"context"

	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ SearchableBlock = (*BackendSearchBlock)(nil)

const defaultBackendSearchBlockPageSize = 2 * 1024 * 1024

type BackendSearchBlock struct {
	id       uuid.UUID
	tenantID string
	l        *local.Backend
}

// NewBackendSearchBlock iterates through the given WAL search data and writes it to the persistent backend
// in a more efficient paged form. Multiple traces are written in the same page to make sure of the flatbuffer
// CreateSharedString feature which dedupes strings across the entire buffer.
func NewBackendSearchBlock(input *StreamingSearchBlock, l *local.Backend, blockID uuid.UUID, tenantID string, enc backend.Encoding, pageSizeBytes int) error {
	var err error
	ctx := context.TODO()
	indexPageSize := 100 * 1024
	kv := &tempofb.KeyValues{} // buffer

	if pageSizeBytes <= 0 {
		pageSizeBytes = defaultBackendSearchBlockPageSize
	}

	// Copy records into the appender
	w, err := newBackendSearchBlockWriter(blockID, tenantID, l, enc)
	if err != nil {
		return err
	}

	a := encoding.NewBufferedAppenderGeneric(w, pageSizeBytes)
	for _, r := range input.appender.Records() {

		// Read
		buf := make([]byte, r.Length)
		_, err = input.file.ReadAt(buf, int64(r.Start))
		if err != nil {
			return err
		}

		s := tempofb.SearchEntryFromBytes(buf)
		data := &tempofb.SearchEntryMutable{
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
	sm := &BlockMeta{
		IndexPageSize: uint32(indexPageSize),
		IndexRecords:  uint32(len(ir)),
		Version:       encoding.LatestEncoding().Version(),
		Encoding:      enc,
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
func (s *BackendSearchBlock) Search(ctx context.Context, p Pipeline, sr *Results) error {
	var pageBuf []byte
	var dataBuf []byte
	var pagesBuf [][]byte
	indexBuf := []common.Record{{}}
	entry := &tempofb.SearchEntry{} // Buffer

	sr.AddBlockInspected()

	meta, err := ReadSearchBlockMeta(ctx, s.l, s.id, s.tenantID)
	if err != nil {
		return err
	}

	vers, err := encoding.FromVersion(meta.Version)
	if err != nil {
		return err
	}

	// Read index
	bmeta := backend.NewBlockMeta(s.tenantID, s.id, meta.Version, meta.Encoding, "")
	cr := backend.NewContextReader(bmeta, "search-index", backend.NewReader(s.l), false)

	ir, err := vers.NewIndexReader(cr, int(meta.IndexPageSize), int(meta.IndexRecords))
	if err != nil {
		return err
	}

	dcr := backend.NewContextReader(bmeta, "search", backend.NewReader(s.l), false)
	dr, err := vers.NewDataReader(dcr, meta.Encoding)
	if err != nil {
		return err
	}

	or := vers.NewObjectReaderWriter()

	i := -1

	for !sr.Quit() {

		i++

		// Next index entry
		record, _ := ir.At(ctx, i)
		if record == nil {
			return nil
		}

		indexBuf[0] = *record
		pagesBuf, pageBuf, err = dr.Read(ctx, indexBuf, pagesBuf, pageBuf)
		if err != nil {
			return err
		}

		pagesBuf[0], _, dataBuf, err = or.UnmarshalAndAdvanceBuffer(pagesBuf[0])
		if err != nil {
			return err
		}

		sr.AddBytesInspected(uint64(len(dataBuf)))

		batch := tempofb.GetRootAsSearchPage(dataBuf, 0)

		// Verify something in the batch matches
		if !p.MatchesTags(batch) {
			// Increment metric still
			sr.AddTraceInspected(uint32(batch.EntriesLength()))
			continue
		}

		l := batch.EntriesLength()
		for j := 0; j < l; j++ {
			sr.AddTraceInspected(1)

			batch.Entries(entry, j)

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
