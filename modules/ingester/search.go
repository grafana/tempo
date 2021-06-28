package ingester

import (
	"bytes"
	"context"
	"fmt"
	"os"

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

type tracefilter func(header *tempofb.SearchData) (matches bool)

type pipeline struct {
	filters []tracefilter
}

func caseInsensitiveContains(s []byte, substr string) bool {
	return bytes.Contains(bytes.ToLower(s), bytes.ToLower([]byte(substr)))
}

func NewSearchPipeline(req *tempopb.SearchRequest) pipeline {
	p := pipeline{}

	if req.RootSpanName != "" {
		/*p.filters = append(p.filters, func(header *tempofb.SearchData) bool {
			return caseInsensitiveContains(header.RootSpanName(), req.RootSpanName)
		})*/
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			return tempofb.SearchDataContains(s, "root.span.name", req.RootSpanName)
		})
	}

	if req.RootAttributeName != "" && req.RootAttributeValue != "" {
		/*p.filters = append(p.filters, func(header *tempofb.TraceHeader) bool {
			kv := &tempofb.KV{}
			for i := 0; i < header.RootSpanProcessTagsLength(); i++ {
				header.RootSpanProcessTags(kv, i)
				if caseInsensitiveContains(kv.Key(), req.RootAttributeName) &&
					caseInsensitiveContains(kv.Value(), req.RootAttributeValue) {
					return true
				}
			}
			return false
		})*/
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

	// Entries in WAL are not paged.
	a := encoding.NewAppender(s)
	if err != nil {
		return nil, err
	}

	s.appender = a

	return s, nil
}

func (s *searchData) Append(ctx context.Context, id common.ID, searchData [][]byte) error {
	data := tempofb.SearchDataMap{}

	kv := &tempofb.KeyValues{}

	// Squash all datas into 1
	for _, sb := range searchData {
		sd := tempofb.SearchDataFromBytes(sb)
		for i := 0; i < sd.DataLength(); i++ {
			sd.Data(kv, i)
			for j := 0; j < kv.ValueLength(); j++ {
				tempofb.SearchDataAppend(data, string(kv.Key()), string(kv.Value(j)))
			}
		}
	}

	return s.appender.Append(id, tempofb.SearchDataFromMap(data))

	/*
		var rnsb []byte

		rst := map[string][]byte{}

		// Squash all separate datas into 1
		for _, s := range searchData {
			b := tempofb.GetRootAsTraceHeader(s, 0)
			if len(b.RootSpanName()) > 0 {
				rnsb = b.RootSpanName()
			}

			kv := &tempofb.KV{}
			for i := 0; i < b.RootSpanProcessTagsLength(); i++ {
				b.RootSpanProcessTags(kv, i)
				rst[string(kv.Key())] = kv.Value()
			}
		}

		b := flatbuffers.NewBuilder(1024)

		rsn := b.CreateByteString(rnsb)

		var rstu []flatbuffers.UOffsetT

		for k, v := range rst {
			ku := b.CreateString(k)
			vu := b.CreateByteString(v)

			tempofb.KVStart(b)
			tempofb.KVAddKey(b, ku)
			tempofb.KVAddValue(b, vu)
			rstu = append(rstu, tempofb.KVEnd(b))
		}

		tempofb.TraceHeaderStartRootSpanProcessTagsVector(b, len(rstu))
		for _, v := range rstu {
			b.PrependUOffsetT(v)
		}
		rstvuu := b.EndVector(len(rstu))

		tempofb.TraceHeaderStart(b)
		tempofb.TraceHeaderAddRootSpanName(b, rsn)
		tempofb.TraceHeaderAddRootSpanProcessTags(b, rstvuu)
		b.Finish(tempofb.TraceHeaderEnd(b))

		return s.appender.Append(id, b.FinishedBytes())*/
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

		//header := tempofb.GetRootAsTraceHeader(buf, 0)
		searchData := tempofb.SearchDataFromBytes(buf)

		if !p.Matches(searchData) {
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
