package search

import (
	"context"
	"os"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ SearchBlock = (*StreamingSearchBlock)(nil)
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
	data := tempofb.SearchDataMutable{}

	kv := &tempofb.KeyValues{}

	// Squash all datas into 1
	for _, sb := range searchData {
		sd := tempofb.SearchDataFromBytes(sb)
		for i := 0; i < sd.TagsLength(); i++ {
			sd.Tags(kv, i)
			for j := 0; j < kv.ValueLength(); j++ {
				data.AddTag(string(kv.Key()), string(kv.Value(j)))
			}
		}
		data.SetStartTimeUnixNano(sd.StartTimeUnixNano())
		data.SetEndTimeUnixNano(sd.EndTimeUnixNano())
	}
	data.TraceID = id

	buf := data.ToBytes()

	return s.appender.Append(id, buf)
}

// Search the streaming block.
func (s *StreamingSearchBlock) Search(ctx context.Context, p Pipeline, sr *SearchResults) error {

	var buf []byte

	rr := s.appender.Records()

	for _, r := range rr {

		if sr.Quit() {
			return nil
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
		sr.AddTraceInspected()

		searchData := tempofb.SearchDataFromBytes(buf)

		if !p.Matches(searchData) {
			continue
		}

		// If we got here then it's a match.
		match := GetSearchResultFromData(searchData)

		if quit := sr.AddResult(ctx, match); quit {
			return nil
		}
	}

	return nil
}
