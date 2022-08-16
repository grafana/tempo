package vparquet

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func (b *backendBlock) open(ctx context.Context) (*parquet.File, *parquet.GenericReader[*Trace], error) {
	rr := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)

	// 128 MB memory buffering
	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), 2*1024*1024, 64)

	pf, err := parquet.OpenFile(br, int64(b.meta.Size))
	if err != nil {
		return nil, nil, err
	}

	r := parquet.NewGenericReader[*Trace](pf) // reader at? parquet file?
	return pf, r, nil
}

func (b *backendBlock) Iterator(ctx context.Context) (Iterator, error) {
	_, r, err := b.open(ctx)
	if err != nil {
		return nil, err
	}

	return &blockIterator{blockID: b.meta.BlockID.String(), r: r}, nil
}

type blockIterator struct {
	blockID string
	r       *parquet.GenericReader[*Trace]
}

func (i *blockIterator) Next(context.Context) (*Trace, error) {
	tr := make([]*Trace, 1)
	_, err := i.r.Read(tr) // jpe batch
	switch err {
	case nil:
		if tr[0] == nil {
			return nil, fmt.Errorf("unexpected nil trace in iteration")
		}
		return tr[0], nil
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
	pool         *rowPool
}

var _ RawIterator = (*rawIterator)(nil)

func (i *rawIterator) getTraceID(r parquet.Row) common.ID {
	for _, v := range r {
		if v.Column() == i.traceIDIndex {
			return v.ByteArray()
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

	if err == io.EOF {
		return nil, nil, nil
	}

	return nil, nil, errors.Wrap(err, fmt.Sprintf("error iterating through block %s", i.blockID))
}

func (i *rawIterator) Close() {
	i.r.Close()
}
