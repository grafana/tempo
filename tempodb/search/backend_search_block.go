package search

import (
	"bytes"
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	tempo_io "github.com/grafana/tempo/pkg/io"
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

	// Pinning specific version instead of latest for safety
	version, err := encoding.FromVersion("v2")
	if err != nil {
		return err
	}

	if pageSizeBytes <= 0 {
		pageSizeBytes = defaultBackendSearchBlockPageSize
	}

	header := tempofb.NewSearchBlockHeaderBuilder()

	w, err := newBackendSearchBlockWriter(blockID, tenantID, l, version, enc)
	if err != nil {
		return err
	}
	a := encoding.NewBufferedAppenderGeneric(w, pageSizeBytes)

	iter, err := input.Iterator()
	if err != nil {
		return errors.Wrap(err, "error getting streaming search block iterator")
	}

	// Copy records into the appender
	for {
		// Read
		id, data, err := iter.Next(ctx)
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "error iterating")
		}

		if id == nil {
			break
		}

		if len(data) == 0 {
			continue
		}

		s := tempofb.SearchEntryFromBytes(data)

		header.AddEntry(s)

		entry := &tempofb.SearchEntryMutable{
			TraceID:           id,
			StartTimeUnixNano: s.StartTimeUnixNano(),
			EndTimeUnixNano:   s.EndTimeUnixNano(),
		}

		for i, l := 0, s.TagsLength(); i < l; i++ {
			s.Tags(kv, i)
			for j, ll := 0, kv.ValueLength(); j < ll; j++ {
				entry.AddTag(string(kv.Key()), string(kv.Value(j)))
			}
		}

		err = a.Append(ctx, id, entry)
		if err != nil {
			return errors.Wrap(err, "error appending to backend block")
		}
	}

	err = a.Complete(ctx)
	if err != nil {
		return err
	}

	// Write index
	ir := a.Records()
	i := version.NewIndexWriter(indexPageSize)
	indexBytes, err := i.Write(ir)
	if err != nil {
		return err
	}
	err = l.Write(ctx, "search-index", backend.KeyPathForBlock(blockID, tenantID), bytes.NewReader(indexBytes), int64(len(indexBytes)), true)
	if err != nil {
		return err
	}

	// Write header
	hb := header.ToBytes()
	err = l.Write(ctx, "search-header", backend.KeyPathForBlock(blockID, tenantID), bytes.NewReader(hb), int64(len(hb)), true)
	if err != nil {
		return err
	}

	// Write meta
	sm := &BlockMeta{
		IndexPageSize: uint32(indexPageSize),
		IndexRecords:  uint32(len(ir)),
		Version:       version.Version(),
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

	// Read header
	// Verify something in the block matches by checking the header
	hbr, hbrlen, err := s.l.Read(ctx, "search-header", backend.KeyPathForBlock(s.id, s.tenantID), true)
	if err != nil {
		return err
	}

	sr.bytesInspected.Add(uint64(hbrlen))

	hb, err := tempo_io.ReadAllWithEstimate(hbr, hbrlen)
	if err != nil {
		return err
	}

	header := tempofb.GetRootAsSearchBlockHeader(hb, 0)
	if !p.MatchesBlock(header) {
		// Block filtered out
		// TODO - metrics ?
		return nil
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

		page := tempofb.GetRootAsSearchPage(dataBuf, 0)
		if !p.MatchesPage(page) {
			// Nothing in the page matches
			// Increment metric still
			sr.AddTraceInspected(uint32(page.EntriesLength()))
			continue
		}

		l := page.EntriesLength()
		for j := 0; j < l; j++ {
			sr.AddTraceInspected(1)

			page.Entries(entry, j)

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
