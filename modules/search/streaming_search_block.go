package search

import (
	"context"
	"os"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var _ SearchBlock = (*StreamingSearchBlock)(nil)

type StreamingSearchBlock struct {
	appender     encoding.Appender
	file         *os.File
	bytesWritten int
}

func (s *StreamingSearchBlock) Clear() error {
	s.file.Close()
	return os.Remove(s.file.Name())
}

func (*StreamingSearchBlock) Complete() error {
	return nil
}

func (s *StreamingSearchBlock) CutPage() (int, error) {
	b := s.bytesWritten
	s.bytesWritten = 0
	return b, nil
}

func (s *StreamingSearchBlock) Write(id common.ID, obj []byte) (int, error) {
	var err error

	_, err = s.file.Write(obj)
	if err != nil {
		return 0, err
	}

	s.bytesWritten += len(obj)

	return len(obj), err
}

var _ common.DataWriter = (*StreamingSearchBlock)(nil)

func NewStreamingSearchBlockForFile(f *os.File) (*StreamingSearchBlock, error) {
	s := &StreamingSearchBlock{
		file: f,
	}

	// Entries are not paged, use non paged appender.
	a := encoding.NewAppender(s)
	s.appender = a

	return s, nil
}

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
		/*if (sd.StartTimeUnixNano() > 0 && sd.StartTimeUnixNano() < minStart) || minStart == 0 {
			minStart = sd.StartTimeUnixNano()
		}
		if sd.EndTimeUnixNano() > 0 && sd.EndTimeUnixNano() > maxEnd {
			maxEnd = sd.EndTimeUnixNano()
		}*/
	}

	buf := data.ToBytes()

	return s.appender.Append(id, buf)
}

func (s *StreamingSearchBlock) Search(ctx context.Context, p Pipeline) ([]*tempopb.TraceSearchMetadata, error) {

	var matches []*tempopb.TraceSearchMetadata

	rr := s.appender.Records()

	for _, r := range rr {

		buf := make([]byte, r.Length)
		_, err := s.file.ReadAt(buf, int64(r.Start))
		if err != nil {
			return nil, err
		}

		//header := tempofb.GetRootAsTraceHeader(buf, 0)
		searchData := tempofb.SearchDataFromBytes(buf)

		if !p.Matches(searchData) {
			continue
		}

		// If we got here then it's a match.
		matches = append(matches, &tempopb.TraceSearchMetadata{
			TraceID:           util.TraceIDToHexString(r.ID),
			RootServiceName:   searchData.Get("root.service.name"),
			RootTraceName:     searchData.Get("root.name"),
			StartTimeUnixNano: searchData.StartTimeUnixNano(),
			DurationMs:        uint32((searchData.EndTimeUnixNano() - searchData.StartTimeUnixNano()) / 1_000_000),
		})

		if len(matches) > 20 {
			break
		}
	}

	return matches, nil
}
