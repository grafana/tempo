package frigg

import (
	"context"
	"time"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

type Backend struct {
	cfg *Config
}

func New(c *Config) *Backend {
	return &Backend{
		cfg: c,
	}
}

func (b *Backend) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}
func (b *Backend) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	return nil, nil
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
