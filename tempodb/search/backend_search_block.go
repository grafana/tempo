package search

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/pkg/errors"

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

//nolint:golint
type SearchDataCombiner struct{}

func (*SearchDataCombiner) Combine(_ string, searchData ...[]byte) ([]byte, bool) {

	if len(searchData) <= 0 {
		return nil, false
	}

	if len(searchData) == 1 {
		return searchData[0], false
	}

	// Squash all datas into 1
	data := tempofb.SearchEntryMutable{}
	kv := &tempofb.KeyValues{} // buffer
	for _, sb := range searchData {
		sd := tempofb.SearchEntryFromBytes(sb)
		for i := 0; i < sd.TagsLength(); i++ {
			sd.Tags(kv, i)
			for j := 0; j < kv.ValueLength(); j++ {
				data.AddTag(string(kv.Key()), string(kv.Value(j)))
			}
		}
		data.SetStartTimeUnixNano(sd.StartTimeUnixNano())
		data.SetEndTimeUnixNano(sd.EndTimeUnixNano())
		data.TraceID = sd.Id()
	}

	return data.ToBytes(), true
}

var _ common.ObjectCombiner = (*SearchDataCombiner)(nil)

//nolint:golint
type SearchDataIterator struct {
	currentIndex int
	records      []common.Record
	file         *os.File
}

func (s *SearchDataIterator) Next(_ context.Context) (common.ID, []byte, error) {
	if s.currentIndex >= len(s.records) {
		return nil, nil, io.EOF
	}

	currentRecord := s.records[s.currentIndex]
	buffer := make([]byte, currentRecord.Length)
	_, err := s.file.ReadAt(buffer, int64(currentRecord.Start))
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading search file")
	}

	s.currentIndex++

	return currentRecord.ID, buffer, nil
}

func (*SearchDataIterator) Close() {
	// file will be closed by StreamingSearchBlock
}

var _ encoding.Iterator = (*SearchDataIterator)(nil)

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

	w, err := newBackendSearchBlockWriter(blockID, tenantID, l, version, enc)
	if err != nil {
		return err
	}

	// set up deduping iterator for streaming search block
	combiner := &SearchDataCombiner{}
	searchIterator := &SearchDataIterator{
		records: input.appender.Records(),
		file:    input.file,
	}
	iter, err := encoding.NewDedupingIterator(searchIterator, combiner, "")
	if err != nil {
		return errors.Wrap(err, "error creating deduping iterator")
	}
	a := encoding.NewBufferedAppenderGeneric(w, pageSizeBytes)

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
		entry := &tempofb.SearchEntryMutable{
			TraceID:           id,
			StartTimeUnixNano: s.StartTimeUnixNano(),
			EndTimeUnixNano:   s.EndTimeUnixNano(),
		}

		l := s.TagsLength()
		for i := 0; i < l; i++ {
			s.Tags(kv, i)
			for j := 0; j < kv.ValueLength(); j++ {
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
