package ingester

import (
	"bytes"
	"context"
	"os"

	flatbuffers "github.com/google/flatbuffers/go"
	tempofb "github.com/grafana/tempo/pkg/tempofb/Tempo"
	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

type blockHeader struct {
	services   map[string]struct{}
	operations map[string][]string
}

type tracefilter func(header *tempofb.TraceHeader) (matches bool)

type pipeline struct {
	filters []tracefilter
}

func NewSearchPipeline(req *tempopb.SearchRequest) pipeline {
	p := pipeline{}

	if req.RootSpanName != "" {
		p.filters = append(p.filters, func(header *tempofb.TraceHeader) bool {
			return bytes.Contains(bytes.ToLower(header.RootSpanName()), bytes.ToLower([]byte(req.RootSpanName)))
		})
	}

	return p
}

func (p *pipeline) Matches(header *tempofb.TraceHeader) bool {

	for _, f := range p.filters {
		if !f(header) {
			return false
		}
	}

	return true
}

func (p *pipeline) MatchesAny(headers []*tempofb.TraceHeader) bool {

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
	instance     *instance
	appender     encoding.Appender
	file         *os.File
	bytesWritten int
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
		instance: i,
	}

	f, err := i.writer.WAL().NewFile(b.BlockID(), i.instanceID, "searchdata")
	if err != nil {
		return nil, err
	}
	s.file = f

	a := encoding.NewAppender(s)
	if err != nil {
		return nil, err
	}

	s.appender = a

	return s, nil
}

func (s *searchData) Append(ctx context.Context, id common.ID, searchData [][]byte) error {
	var rnsb []byte

	// Squash all separate datas into 1
	for _, s := range searchData {
		b := tempofb.GetRootAsTraceHeader(s, 0)
		if len(b.RootSpanName()) > 0 {
			rnsb = b.RootSpanName()
		}
	}

	b := flatbuffers.NewBuilder(1024)

	rns := b.CreateByteString(rnsb)

	tempofb.TraceHeaderStart(b)
	tempofb.TraceHeaderAddRootSpanName(b, rns)

	b.Finish(tempofb.TraceHeaderEnd(b))

	return s.appender.Append(id, b.FinishedBytes())
}

func (s *searchData) Search(ctx context.Context, p pipeline) ([]common.ID, error) {

	var matches []common.ID

	rr := s.appender.Records()

	for _, r := range rr {

		buf := make([]byte, r.Length)
		_, err := s.file.ReadAt(buf, int64(r.Start))
		if err != nil {
			return nil, err
		}

		header := tempofb.GetRootAsTraceHeader(buf, 0)

		if !p.Matches(header) {
			continue
		}

		// If we got here then it's a match.
		matches = append(matches, r.ID)

		if len(matches) > 20 {
			break
		}
	}

	return matches, nil
}
