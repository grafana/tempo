package search

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const defaultBackendSearchBlockPageSize = 2 * 1024 * 1024

type BackendSearchBlock struct {
	id       uuid.UUID
	tenantID string
	r        backend.Reader
}

// NewBackendSearchBlock iterates through the given WAL search data and writes it to the persistent backend
// in a more efficient paged form. Multiple traces are written in the same page to make sure of the flatbuffer
// CreateSharedString feature which dedupes strings across the entire buffer.
func NewBackendSearchBlock(input *StreamingSearchBlock, rw backend.Writer, blockID uuid.UUID, tenantID string, enc backend.Encoding, pageSizeBytes int) error {
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

	header := tempofb.NewSearchBlockHeaderMutable()

	w, err := newBackendSearchBlockWriter(blockID, tenantID, rw, version, enc)
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
	err = rw.Write(ctx, "search-index", blockID, tenantID, indexBytes, true)
	if err != nil {
		return err
	}

	// Write header
	hb := header.ToBytes()
	err = rw.Write(ctx, "search-header", blockID, tenantID, hb, true)
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
	return WriteSearchBlockMeta(ctx, rw, blockID, tenantID, sm)
}

// OpenBackendSearchBlock opens the search data for an existing block in the given backend.
func OpenBackendSearchBlock(blockID uuid.UUID, tenantID string, r backend.Reader) *BackendSearchBlock {
	return &BackendSearchBlock{
		id:       blockID,
		tenantID: tenantID,
		r:        r,
	}
}

// BlockID provides access to the private field id
func (s *BackendSearchBlock) BlockID() uuid.UUID {
	return s.id
}

func (s *BackendSearchBlock) Tags(ctx context.Context, tags map[string]struct{}) error {
	header, err := s.readSearchHeader(ctx)
	if err != nil {
		return err
	}

	kv := &tempofb.KeyValues{}
	for i, ii := 0, header.TagsLength(); i < ii; i++ {
		header.Tags(kv, i)
		key := string(kv.Key())
		// check the tag is already set, this is more performant with repetitive values
		if _, ok := tags[key]; !ok {
			tags[key] = struct{}{}
		}
	}

	return nil
}

func (s *BackendSearchBlock) TagValues(ctx context.Context, tagName string, tagValues map[string]struct{}) error {
	header, err := s.readSearchHeader(ctx)
	if err != nil {
		return err
	}

	kv := tempofb.FindTag(header, &tempofb.KeyValues{}, []byte(tagName))
	if kv != nil {
		for j, valueLength := 0, kv.ValueLength(); j < valueLength; j++ {
			value := string(kv.Value(j))
			// check the value is already set, this is more performant with repetitive values
			if _, ok := tagValues[value]; !ok {
				tagValues[value] = struct{}{}
			}
		}
	}
	return nil
}

// Search iterates through the block looking for matches.
func (s *BackendSearchBlock) Search(ctx context.Context, p Pipeline, sr *Results) error {
	var pageBuf []byte
	var dataBuf []byte
	var pagesBuf [][]byte
	indexBuf := []common.Record{{}}
	entry := &tempofb.SearchEntry{} // Buffer

	meta, err := ReadSearchBlockMeta(ctx, s.r, s.id, s.tenantID)
	if err != nil {
		if err == backend.ErrDoesNotExist {
			// This means one of the following:
			// * This block predates search and doesn't actually have
			//   search data (we create the block entry regardless)
			// * This block is deleted between when the search was
			//   initiated and when we got here.
			// In either case it is not an error.
			return nil
		}
		return err
	}

	vers, err := encoding.FromVersion(meta.Version)
	if err != nil {
		return err
	}

	// Read header
	// Verify something in the block matches by checking the header
	hb, err := s.r.Read(ctx, "search-header", s.id, s.tenantID, true)
	if err != nil {
		return err
	}

	sr.bytesInspected.Add(uint64(len(hb)))

	header := tempofb.GetRootAsSearchBlockHeader(hb, 0)
	if !p.MatchesBlock(header) {
		// Block filtered out
		sr.AddBlockSkipped()
		return nil
	}

	sr.AddBlockInspected()

	// Read index
	bmeta := backend.NewBlockMeta(s.tenantID, s.id, meta.Version, meta.Encoding, "")
	cr := backend.NewContextReader(bmeta, "search-index", s.r, false)

	ir, err := vers.NewIndexReader(cr, int(meta.IndexPageSize), int(meta.IndexRecords))
	if err != nil {
		return err
	}

	dcr := backend.NewContextReader(bmeta, "search", s.r, false)
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

func (s *BackendSearchBlock) readSearchHeader(ctx context.Context) (*tempofb.SearchBlockHeader, error) {
	hb, err := s.r.Read(ctx, "search-header", s.id, s.tenantID, true)
	if err != nil {
		return nil, err
	}
	return tempofb.GetRootAsSearchBlockHeader(hb, 0), nil
}
