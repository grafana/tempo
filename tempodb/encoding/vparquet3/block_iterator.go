package vparquet3

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func (b *backendBlock) open(ctx context.Context) (*parquet.File, *parquet.Reader, error) { //nolint:all //deprecated
	rr := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta)

	pf, err := parquet.OpenFile(rr, int64(b.meta.Size), parquet.SkipBloomFilters(true), parquet.SkipPageIndex(true), parquet.ReadBufferSize(4*1024*1024))
	if err != nil {
		return nil, nil, err
	}

	r := parquet.NewReader(pf, parquet.SchemaOf(&Trace{}))
	return pf, r, nil
}

func (b *backendBlock) rawIter(ctx context.Context, pool *rowPool) (*rawIterator, error) {
	pf, r, err := b.open(ctx)
	if err != nil {
		return nil, err
	}

	traceIDIndex, _ := parquetquery.GetColumnIndexByPath(pf, TraceIDColumnName)
	if traceIDIndex < 0 {
		return nil, fmt.Errorf("cannot find trace ID column in '%s' in block '%s'", TraceIDColumnName, b.meta.BlockID.String())
	}

	return &rawIterator{b.meta.BlockID.String(), r, traceIDIndex, pool}, nil
}

type rawIterator struct {
	blockID      string
	r            *parquet.Reader //nolint:all //deprecated
	traceIDIndex int
	pool         *rowPool
}

var _ RawIterator = (*rawIterator)(nil)

func (i *rawIterator) getTraceID(r parquet.Row) common.ID {
	for _, v := range r {
		if v.Column() == i.traceIDIndex {
			// Important - clone to get a detached copy that lives outside the pool.
			return v.Clone().ByteArray()
		}
	}
	return nil
}

func (i *rawIterator) Next(context.Context) (common.ID, parquet.Row, error) {
	rows := []parquet.Row{i.pool.Get()}
	n, err := i.r.ReadRows(rows)
	if n > 0 {
		return i.getTraceID(rows[0]), rows[0], nil
	}

	if errors.Is(err, io.EOF) {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, fmt.Errorf("error iterating through block %s: %w", i.blockID, err)
	}
	return nil, nil, nil
}

func (i *rawIterator) peekNextID(context.Context) (common.ID, error) { // nolint:unused // this is required to satisfy the bookmarkIterator interface
	return nil, common.ErrUnsupported
}

func (i *rawIterator) Close() {
	i.r.Close()
}
