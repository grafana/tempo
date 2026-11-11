package vparquet5

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/v3/pkg/parquetquery"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

// otherTypeReads counts the values a fetch reads from generic attributes for attributes with a dedicated column.
type otherTypeReads struct {
	values atomic.Uint64
}

// wrap returns iter counting into r. A nil r leaves iter as is.
func (r *otherTypeReads) wrap(iter parquetquery.Iterator) parquetquery.Iterator {
	if r == nil || iter == nil {
		return iter
	}
	return &otherTypeIterator{Iterator: iter, reads: r}
}

// report adds the values read since the last report, if any, as an event on ctx's span.
// ctx is the caller's, since the fetch span ends before the results are read.
func (r *otherTypeReads) report(ctx context.Context, meta *backend.BlockMeta) {
	n := r.values.Swap(0)
	if n == 0 {
		return
	}
	trace.SpanFromContext(ctx).AddEvent("dedicated column values of another type", trace.WithAttributes(
		attribute.String("tenantID", meta.TenantID),
		attribute.String("blockID", meta.BlockID.String()),
		attribute.Int64("values", int64(n)),
	))
}

// otherTypeIterator counts the rows the generic half of a dedicated existence iterator returns.
type otherTypeIterator struct {
	parquetquery.Iterator
	reads *otherTypeReads
	last  parquetquery.RowNumber
	seen  bool
}

func (i *otherTypeIterator) Next() (*parquetquery.IteratorResult, error) {
	res, err := i.Iterator.Next()
	i.count(res)
	return res, err
}

func (i *otherTypeIterator) SeekTo(t parquetquery.RowNumber, definitionLevel int) (*parquetquery.IteratorResult, error) {
	res, err := i.Iterator.SeekTo(t, definitionLevel)
	i.count(res)
	return res, err
}

// count skips a row it already counted, since SeekTo returns the current row again when it is past t.
func (i *otherTypeIterator) count(res *parquetquery.IteratorResult) {
	if res == nil || (i.seen && res.RowNumber == i.last) {
		return
	}
	i.last, i.seen = res.RowNumber, true
	i.reads.values.Add(1)
}

// reportingIterator reports reads when the fetch's results are closed, after the query consumed them.
type reportingIterator[T any] struct {
	traceql.CommonIterator[T]
	report func()
}

func (i *reportingIterator[T]) Close() {
	i.CommonIterator.Close()
	i.report()
}
