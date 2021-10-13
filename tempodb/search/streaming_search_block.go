package search

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ SearchableBlock = (*StreamingSearchBlock)(nil)

// StreamingSearchBlock is search data that is read/write, i.e. for traces in the WAL.
type StreamingSearchBlock struct {
	BlockID        uuid.UUID // todo: add the full meta?
	appender       encoding.Appender
	file           *os.File
	closed         atomic.Bool
	bufferedWriter *bufio.Writer
	header         *tempofb.SearchBlockHeaderMutable
	v              encoding.VersionedEncoding
	enc            backend.Encoding
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
func NewStreamingSearchBlockForFile(f *os.File, bufferedWriter *bufio.Writer, version string, enc backend.Encoding) (*StreamingSearchBlock, error) {
	v, err := encoding.FromVersion(version)
	if err != nil {
		return nil, err
	}
	s := &StreamingSearchBlock{
		file:   f,
		header: tempofb.NewSearchBlockHeaderMutable(),
		v:      v,
		enc:    enc,
	}
	s.bufferedWriter = bufferedWriter

	// Use versioned encoding to create paged entries
	dataWriter, err := s.v.NewDataWriter(s.bufferedWriter, enc)
	if err != nil {
		return nil, err
	}

	a := encoding.NewAppender(dataWriter)
	s.appender = a

	return s, nil
}

// Append the given search data to the streaming block. Multiple byte buffers of search data for
// the same trace can be passed and are merged into one entry.
// After calling Append FlushBuffer() must be called to guarantee that all data makes it to the disk
func (s *StreamingSearchBlock) Append(ctx context.Context, id common.ID, searchData [][]byte) error {
	combined, _ := staticCombiner.Combine("", searchData...)

	if len(combined) <= 0 {
		return nil
	}

	s.header.AddEntry(tempofb.SearchEntryFromBytes(combined))

	return s.appender.Append(id, combined)
}

// FlushBuffer force flushes all buffered data to disk. This must be called after Append() to guarantee
// that all data makes it to the disk. It is intended that there are many Append() calls per FlushBuffer().
// It must also be called before any attempts to use appender records to read the file such as in Search() or
// Iterator().
func (s *StreamingSearchBlock) FlushBuffer() error {
	if s.bufferedWriter == nil {
		return nil
	}
	return s.bufferedWriter.Flush()
}

// Search the streaming block.
func (s *StreamingSearchBlock) Search(ctx context.Context, p Pipeline, sr *Results) error {
	if s.closed.Load() {
		// Generally this means block has already been deleted
		return nil
	}

	if !p.MatchesBlock(s.header) {
		sr.AddBlockSkipped()
		return nil
	}

	sr.AddBlockInspected()

	// calling s.Iterator() forces a buffer flush. no need to do it before here
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

		entry := tempofb.SearchEntryFromBytes(obj)

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
	// flush buffers to make sure all data is on disk
	err := s.FlushBuffer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to flush buffer of search streaming block")
	}

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
