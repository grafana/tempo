package ingester

import (
	"context"
	"fmt"
	"os"
	"time"

	tempofb "github.com/grafana/tempo/pkg/tempofb/Tempo"
	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

type SearchBlock interface {
	Search(ctx context.Context, p pipeline) ([]*tempopb.TraceSearchMetadata, error)
}

var _ SearchBlock = (*searchData)(nil)

type blockHeader struct {
	services   map[string]struct{}
	operations map[string][]string
}

type tracefilter func(header *tempofb.SearchData) (matches bool)

type pipeline struct {
	filters []tracefilter
}

func NewSearchPipeline(req *tempopb.SearchRequest) pipeline {
	p := pipeline{}

	// Obsolete
	if req.RootSpanName != "" {
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			return tempofb.SearchDataContains(s, "root.span.name", req.RootSpanName)
		})
	}

	// Obsolete
	if req.RootAttributeName != "" && req.RootAttributeValue != "" {
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			return tempofb.SearchDataContains(s, fmt.Sprint("root.span.", req.RootAttributeName), req.RootAttributeValue)
		})
	}

	if len(req.Tags) > 0 {
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			// Must match all
			for k, v := range req.Tags {
				if !tempofb.SearchDataContains(s, k, v) {
					return false
				}
			}
			return true
		})
	}

	if req.MinDurationMs > 0 {
		minDuration := req.MinDurationMs * uint32(time.Millisecond)
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			return (s.EndTimeUnixNano()-s.StartTimeUnixNano())*uint64(time.Nanosecond) >= uint64(minDuration)
		})
	}

	if req.MaxDurationMs > 0 {
		maxDuration := req.MaxDurationMs * uint32(time.Millisecond)
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			return (s.EndTimeUnixNano()-s.StartTimeUnixNano())*uint64(time.Nanosecond) <= uint64(maxDuration)
		})
	}

	return p
}

func (p *pipeline) Matches(header *tempofb.SearchData) bool {

	for _, f := range p.filters {
		if !f(header) {
			return false
		}
	}

	return true
}

func (p *pipeline) MatchesAny(headers []*tempofb.SearchData) bool {

	if len(p.filters) == 0 {
		// Empty pipeline matches everything
		return true
	}

	for _, h := range headers {
		for _, f := range p.filters {
			if f(h) {
				return true
			}
		}
	}

	return false
}

type searchData struct {
	//instance     *instance
	appender     encoding.Appender
	file         *os.File
	bytesWritten int
}

func (s *searchData) Clear() error {
	s.file.Close()
	return os.Remove(s.file.Name())
}

func (*searchData) Complete() error {
	return nil
}

func (s *searchData) CutPage() (int, error) {
	b := s.bytesWritten
	s.bytesWritten = 0
	return b, nil
}

func (s *searchData) Write(id common.ID, obj []byte) (int, error) {
	var err error

	_, err = s.file.Write(obj)
	if err != nil {
		return 0, err
	}

	s.bytesWritten += len(obj)

	return len(obj), err
}

var _ common.DataWriter = (*searchData)(nil)

func NewSearchDataForAppendBlock(i *instance, b *wal.AppendBlock) (*searchData, error) {
	s := &searchData{
		//instance: i,
	}

	f, err := i.writer.WAL().NewFile(b.BlockID(), i.instanceID, "searchdata")
	if err != nil {
		return nil, err
	}
	s.file = f

	// Entries in WAL are not paged.
	a := encoding.NewAppender(s)
	if err != nil {
		return nil, err
	}

	s.appender = a

	return s, nil
}

func (s *searchData) Append(ctx context.Context, t *trace) error {
	data := tempofb.SearchDataMap{}

	var minStart uint64
	var maxEnd uint64

	kv := &tempofb.KeyValues{}

	// Squash all datas into 1
	for _, sb := range t.searchData {
		sd := tempofb.SearchDataFromBytes(sb)
		for i := 0; i < sd.TagsLength(); i++ {
			sd.Tags(kv, i)
			for j := 0; j < kv.ValueLength(); j++ {
				tempofb.SearchDataAppend(data, string(kv.Key()), string(kv.Value(j)))
			}
		}
		if (sd.StartTimeUnixNano() > 0 && sd.StartTimeUnixNano() < minStart) || minStart == 0 {
			minStart = sd.StartTimeUnixNano()
		}
		if sd.EndTimeUnixNano() > 0 && sd.EndTimeUnixNano() > maxEnd {
			maxEnd = sd.EndTimeUnixNano()
		}
	}

	buf := tempofb.SearchDataBytesFromValues(t.traceID, data, minStart, maxEnd)

	return s.appender.Append(t.traceID, buf)
}

func (s *searchData) Search(ctx context.Context, p pipeline) ([]*tempopb.TraceSearchMetadata, error) {

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
			TraceID:           r.ID,
			RootServiceName:   tempofb.SearchDataGet(searchData, "root.service.name"),
			RootTraceName:     tempofb.SearchDataGet(searchData, "root.name"),
			StartTimeUnixNano: searchData.StartTimeUnixNano(),
			DurationMs:        uint32((searchData.EndTimeUnixNano() - searchData.StartTimeUnixNano()) / 1_000_000),
		})

		if len(matches) > 20 {
			break
		}
	}

	return matches, nil
}
