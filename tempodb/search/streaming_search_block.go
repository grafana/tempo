package search

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// StreamingSearchBlock is search data that is read/write, i.e. for traces in the WAL.
type StreamingSearchBlock struct {
	blockID   uuid.UUID // todo: add the full meta?
	appender  encoding.Appender
	file      *os.File
	closed    atomic.Bool
	header    *tempofb.SearchBlockHeaderMutable
	headerMtx sync.RWMutex
	v         encoding.VersionedEncoding
	enc       backend.Encoding
}

// Close closes the WAL file. Used in tests
func (s *StreamingSearchBlock) Close() error {
	s.closed.Store(true)
	return s.file.Close()
}

// Clear deletes the files for this block.
func (s *StreamingSearchBlock) Clear() error {
	s.closed.Store(true)
	s.file.Close()
	return os.Remove(s.file.Name())
}

// NewStreamingSearchBlockForFile creates a new streaming block that will read/write the given file.
// File must be opened for read/write permissions.
func NewStreamingSearchBlockForFile(f *os.File, blockID uuid.UUID, version string, enc backend.Encoding) (*StreamingSearchBlock, error) {
	v, err := encoding.FromVersion(version)
	if err != nil {
		return nil, err
	}
	s := &StreamingSearchBlock{
		blockID: blockID,
		file:    f,
		header:  tempofb.NewSearchBlockHeaderMutable(),
		v:       v,
		enc:     enc,
	}

	// Use versioned encoding to create paged entries
	dataWriter, err := s.v.NewDataWriter(f, enc)
	if err != nil {
		return nil, err
	}

	a := encoding.NewAppender(dataWriter)
	s.appender = a

	return s, nil
}

// BlockID provides access to the private field blockID
func (s *StreamingSearchBlock) BlockID() uuid.UUID {
	return s.blockID
}

// Append the given search data to the streaming block. Multiple byte buffers of search data for
// the same trace can be passed and are merged into one entry.
func (s *StreamingSearchBlock) Append(ctx context.Context, id common.ID, searchData [][]byte) error {
	combined, _, err := staticCombiner.Combine("", searchData...)
	if err != nil {
		return fmt.Errorf("error combining: %w", err)
	}

	if len(combined) <= 0 {
		return nil
	}

	s.headerMtx.Lock()
	s.header.AddEntry(tempofb.NewSearchEntryFromBytes(combined))
	s.headerMtx.Unlock()

	return s.appender.Append(id, combined)
}

func (s *StreamingSearchBlock) Tags(ctx context.Context, tags map[string]struct{}) error {
	s.header.Tags.RangeKeys(func(k string) {
		// check the tag is already set, this is more performant with repetitive values
		if _, ok := tags[k]; !ok {
			tags[k] = struct{}{}
		}
	})
	return nil
}

func (s *StreamingSearchBlock) TagValues(ctx context.Context, tagName string, tagValues map[string]struct{}) error {
	s.header.Tags.RangeKeyValues(tagName, func(v string) {
		// check the value is already set, this is more performant with repetitive values
		if _, ok := tagValues[v]; !ok {
			tagValues[v] = struct{}{}
		}
	})
	return nil
}

// Search the streaming block.
func (s *StreamingSearchBlock) Search(ctx context.Context, p Pipeline, sr *Results) error {
	entry := &tempofb.SearchEntry{}

	if s.closed.Load() {
		// Generally this means block has already been deleted
		return nil
	}

	s.headerMtx.RLock()
	matched := p.MatchesBlock(s.header)
	s.headerMtx.RUnlock()
	if !matched {
		sr.AddBlockSkipped()
		return nil
	}

	sr.AddBlockInspected()

	iter, err := s.Iterator()
	if err != nil {
		return err
	}
	defer iter.Close()

	for {

		if sr.Quit() {
			return nil
		}

		_, obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		sr.AddBytesInspected(uint64(len(obj)))
		sr.AddTraceInspected(1)

		entry.Reset(obj)

		if !p.Matches(entry) {
			continue
		}

		// If we got here then it's a match.
		match := GetSearchResultFromData(entry)

		if quit := sr.AddResult(ctx, match); quit {
			return nil
		}
	}

	return nil
}

func (s *StreamingSearchBlock) Iterator() (encoding.Iterator, error) {
	iter := &streamingSearchBlockIterator{
		records:     s.appender.Records(),
		file:        s.file,
		pagesBuffer: make([][]byte, 1),
	}

	dr, err := s.v.NewDataReader(backend.NewContextReaderWithAllReader(s.file), s.enc)
	if err != nil {
		return nil, err
	}
	iter.dataReader = dr

	iter.objectRW = s.v.NewObjectReaderWriter()

	combiner := &DataCombiner{}

	// Streaming (wal) blocks have to be deduped.
	return encoding.NewDedupingIterator(iter, combiner, "")
}

type streamingSearchBlockIterator struct {
	currentIndex int
	records      []common.Record
	file         *os.File
	dataReader   common.DataReader
	objectRW     common.ObjectReaderWriter

	pagesBuffer [][]byte
}

var _ encoding.Iterator = (*streamingSearchBlockIterator)(nil)

func (s *streamingSearchBlockIterator) Next(ctx context.Context) (common.ID, []byte, error) {
	if s.currentIndex >= len(s.records) {
		return nil, nil, io.EOF
	}

	currentRecord := s.records[s.currentIndex]

	// Use unique buffer that can be returned to the caller.
	// This is primarily for DedupingIterator which uses 2 buffers at once.
	var buffer []byte
	s.pagesBuffer[0] = make([]byte, currentRecord.Length)
	pagesBuffer, _, err := s.dataReader.Read(ctx, []common.Record{currentRecord}, s.pagesBuffer, buffer)
	if err != nil {
		return nil, nil, err
	}

	_, pagesBuffer[0], err = s.objectRW.UnmarshalObjectFromReader(bytes.NewReader(pagesBuffer[0]))
	if err != nil {
		return nil, nil, err
	}

	s.currentIndex++

	return currentRecord.ID, pagesBuffer[0], nil
}

func (*streamingSearchBlockIterator) Close() {
	// file will be closed by StreamingSearchBlock
}
