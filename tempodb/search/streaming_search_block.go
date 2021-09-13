package search

import (
	"context"
	"os"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ SearchableBlock = (*StreamingSearchBlock)(nil)
var _ common.DataWriter = (*StreamingSearchBlock)(nil)

// StreamingSearchBlock is search data that is read/write, i.e. for traces in the WAL.
type StreamingSearchBlock struct {
	appender     encoding.Appender
	file         *os.File
	bytesWritten int
}

// Clear deletes the files for this block.
func (s *StreamingSearchBlock) Clear() error {
	s.file.Close()
	return os.Remove(s.file.Name())
}

func (*StreamingSearchBlock) Complete() error {
	return nil
}

// CutPage returns the number of bytes written previously so that the appender can build the index.
func (s *StreamingSearchBlock) CutPage() (int, error) {
	b := s.bytesWritten
	s.bytesWritten = 0
	return b, nil
}

// Write the entry to the end of the file. The number of bytes written is saved and returned through CutPage.
func (s *StreamingSearchBlock) Write(id common.ID, obj []byte) (int, error) {
	var err error

	_, err = s.file.Write(obj)
	if err != nil {
		return 0, err
	}

	s.bytesWritten += len(obj)

	return len(obj), err
}

// NewStreamingSearchBlockForFile creates a new streaming block that will read/write the given file.
// File must be opened for read/write permissions.
func NewStreamingSearchBlockForFile(f *os.File) (*StreamingSearchBlock, error) {
	s := &StreamingSearchBlock{
		file: f,
	}

	// Entries are not paged, use non paged appender.
	a := encoding.NewAppender(s)
	s.appender = a

	return s, nil
}

// Append the given search data to the streaming block. Multiple byte buffers of search data for
// the same trace can be passed and are merged into one entry.
func (s *StreamingSearchBlock) Append(ctx context.Context, id common.ID, searchData [][]byte) error {
	combined, _ := (&SearchDataCombiner{}).Combine("", searchData...)
	return s.appender.Append(id, combined)
}

// Search the streaming block.
func (s *StreamingSearchBlock) Search(ctx context.Context, p Pipeline, sr *Results) error {

	var buf []byte

	sr.AddBlockInspected()

	rr := s.appender.Records()

	for _, r := range rr {

		if sr.Quit() {
			return nil
		}

		if r.Length == 0 {
			continue
		}

		// Reset/resize buffer
		if cap(buf) < int(r.Length) {
			buf = make([]byte, r.Length)
		}
		buf = buf[:r.Length]

		_, err := s.file.ReadAt(buf, int64(r.Start))
		if err != nil {
			return err
		}

		sr.AddBytesInspected(uint64(r.Length))
		sr.AddTraceInspected(1)

		entry := tempofb.SearchEntryFromBytes(buf)

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
