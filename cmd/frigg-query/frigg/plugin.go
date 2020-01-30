package frigg

import (
	"context"
	"encoding/binary"
	"log"
	"time"

	"github.com/grafana/frigg/pkg/friggpb"

	"google.golang.org/grpc"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

type Backend struct {
	conn   *grpc.ClientConn
	client friggpb.QuerierClient
}

func New(cfg *Config) *Backend {
	conn, err := grpc.Dial("192.168.1.126:3100", grpc.WithInsecure()) // jpe : how to pass config?
	if err != nil {
		log.Panic(err)
	}

	return &Backend{
		conn:   conn,
		client: friggpb.NewQuerierClient(conn),
	}
}

func (b *Backend) Close() {
	b.conn.Close()
}

func (b *Backend) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}
func (b *Backend) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	traceIDBytes := make([]byte, 16)
	binary.BigEndian.PutUint64(traceIDBytes[:8], traceID.High)
	binary.BigEndian.PutUint64(traceIDBytes[8:], traceID.Low)

	_, err := b.client.FindTraceByID(ctx, &friggpb.TraceByIDRequest{
		TraceID: traceIDBytes,
	})
	if err != nil {
		return nil, err
	}

	return &model.Trace{
		Spans:      []*model.Span{},
		ProcessMap: []model.Trace_ProcessMapping{},
	}, nil
}
func (b *Backend) GetServices(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (b *Backend) GetOperations(ctx context.Context, query spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	return nil, nil
}
func (b *Backend) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	return nil, nil
}
func (b *Backend) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	return nil, nil
}
func (b *Backend) WriteSpan(span *model.Span) error {
	return nil
}
