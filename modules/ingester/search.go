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

/*type traceHeader struct {
	rootSpanName         string
	rootSpanProcessTags  map[string]string
	startTime            time.Time
	endTime              time.Time
	duration             int
	otherSpanNames       []string
	otherSpanProcessTags map[string][]string
	hasError             bool
}*/

/*type searchReq struct {
	tenantID    string
	service     string
	operation   string
	maxResults  uint
	minDuration time.Duration
	maxDuration time.Duration
	from        time.Time
	through     time.Time
}*/

type tracefilter func(req *tempopb.SearchRequest, header *tempofb.TraceHeader) (matches bool)

var _globalPipeline = []tracefilter{
	// Root span name contains
	func(req *tempopb.SearchRequest, header *tempofb.TraceHeader) (matches bool) {
		// Root span name filter specified?
		if req.RootSpanName != "" {
			return bytes.Contains(bytes.ToLower(header.RootSpanName()), bytes.ToLower([]byte(req.RootSpanName)))
		}
		return true
	},
}

type searchData struct {
	instance *instance
	//name     string
	//blockID  uuid.UUID
	//blockHeader blockHeader
	//tracker      backend.AppendTracker
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
	//s.tracker, err = s.instance.local.Append(context.TODO(), s.name, s.blockID, s.instance.instanceID, s.tracker, obj)

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

func (s *searchData) Append(ctx context.Context, id common.ID, header *tempopb.TraceHeader) error {

	b := flatbuffers.NewBuilder(1024)

	rns := b.CreateString(header.RootSpanName)

	tempofb.TraceHeaderStart(b)
	tempofb.TraceHeaderAddRootSpanName(b, rns)

	b.Finish(tempofb.TraceHeaderEnd(b))

	return s.appender.Append(id, b.FinishedBytes())
}

func (s *searchData) Search(ctx context.Context, req *tempopb.SearchRequest) ([]common.ID, error) {

	var matches []common.ID

	rr := s.appender.Records()

	for _, r := range rr {

		buf := make([]byte, r.Length)
		_, err := s.file.ReadAt(buf, int64(r.Start))
		if err != nil {
			return nil, err
		}

		header := tempofb.GetRootAsTraceHeader(buf, 0)

		for _, f := range _globalPipeline {
			if !f(req, header) {
				break
			}
		}

		// If we got here then it's a match.
		matches = append(matches, r.ID)

		if len(matches) > 20 {
			break
		}
	}

	return matches, nil
}

/*func (i *instance) recordTraceHeader() {

}

func (i *instance) search(req searchReq) ([]*trace, error) {

	// Search traces in memory
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	var matches []*trace

	for h, t := range i.traceHeaders {
		if req.service != "" && t.rootSpanProcessTags["service.name"] != req.service {
			continue
		}

		// Matches
		matches = append(matches, i.traces[h])

		if len(matches) >= int(req.maxResults) {
			break
		}
	}

	return matches, nil
}*/
