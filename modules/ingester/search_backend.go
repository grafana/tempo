package ingester

/*
import (
	"context"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type searchDataBackend struct {
	b *encoding.BackendBlock
}

var _ SearchBlock = (*searchDataBackend)(nil)

type searchWriterBuffered struct {
	tracker backend.AppendTracker
	a       encoding.Appender
	builder *flatbuffers.Builder
	page    flatbuffers.UOffsetT
	entries []flatbuffers.UOffsetT
}

var _ common.DataWriter = (*searchWriterBuffered)(nil)

func NewSearchWriterBuffered() *searchWriterBuffered {
	return &searchWriterBuffered{
		builder: flatbuffers.NewBuilder(1024),
	}
}

func (*searchWriterBuffered) Complete() error {
	return nil
}

func (s *searchWriterBuffered) CutPage() (int, error) {

	s.builder.Finish(s.page)
	buf := s.builder.FinishedBytes()

	s.

	s.builder.Reset()

	s.entries = s.entries[:0]

	return b, nil
}

func (s *searchWriterBuffered) Write(id common.ID, obj []byte) (int, error) {
	var err error

	return len(obj), err
}

func SearchDataFromBlock(b *encoding.BackendBlock) error {
	// TODO - Open existing search data files
	return nil
}

func CompleteSearchDataForBlock(b *encoding.BackendBlock) error {
	s := &searchDataBackend{}

	app := encoding.NewBufferedAppender(NewSearchWriterBuffered(), )
}

func (s *searchDataBackend) Search(ctx context.Context, p pipeline) ([]common.ID, error) {
	return nil, nil
}

*/
