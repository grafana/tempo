package vparquet

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/parquetquery"
)

type blockIterator struct {
	blockID string
	r       *parquet.Reader
	//pool    *sync.Pool
	pool *slicePool
}

func (b *backendBlock) Iterator(ctx context.Context, pool *slicePool) (Iterator, error) {
	rr := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)

	// 32 MB memory buffering
	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), 512*1024, 64)

	pf, err := parquet.OpenFile(br, int64(b.meta.Size))
	if err != nil {
		return nil, err
	}

	r := parquet.NewReader(pf, parquet.SchemaOf(&Trace{}))

	return &blockIterator{blockID: b.meta.BlockID.String(), r: r, pool: pool}, nil
}

func (b *backendBlock) RawIterator(ctx context.Context) (*rawIterator, error) {
	rr := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)

	// 32 MB memory buffering
	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), 512*1024, 64)

	pf, err := parquet.OpenFile(br, int64(b.meta.Size))
	if err != nil {
		return nil, err
	}

	r := parquet.NewReader(pf, parquet.SchemaOf(&Trace{}))

	traceIDIndex, _ := parquetquery.GetColumnIndexByPath(pf, TraceIDColumnName)

	return &rawIterator{blockID: b.meta.BlockID.String(), r: r, traceIDIndex: traceIDIndex}, nil
}

func (i *blockIterator) Next(context.Context) (*Trace, error) {
	//var t *Trace
	t := i.pool.Get()

	switch err := i.r.Read(t); err {
	case nil:
		return t, nil
	case io.EOF:
		return nil, nil
	default:
		return nil, errors.Wrap(err, fmt.Sprintf("error iterating through block %s", i.blockID))
	}
}

func (i *blockIterator) Close() {
	// parquet reader is shared, lets not close it here
}

type rawIterator struct {
	blockID      string
	r            *parquet.Reader
	traceIDIndex int
}

func (i *rawIterator) getTraceID(r parquet.Row) []byte {
	for _, v := range r {
		if v.Column() == i.traceIDIndex {
			return v.ByteArray()
		}
	}
	return nil
}

func (i *rawIterator) Next(context.Context) ([]byte, parquet.Row, error) {
	rows := make([]parquet.Row, 1)

	n, err := i.r.ReadRows(rows)
	if n > 0 {
		return i.getTraceID(rows[0]), rows[0], nil
	}

	if err == io.EOF {
		return nil, nil, nil
	}

	return nil, nil, errors.Wrap(err, fmt.Sprintf("error iterating through block %s", i.blockID))
}
