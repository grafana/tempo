package vparquet

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
)

type blockIterator struct {
	blockID string
	r       *parquet.Reader
	pool    *sync.Pool
}

func (b *backendBlock) Iterator(ctx context.Context, pool *sync.Pool) (Iterator, error) {
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

func (i *blockIterator) Next(context.Context) (*Trace, error) {
	var t *Trace
	x := i.pool.Get()
	if x != nil {
		if cast, ok := x.(*Trace); ok {
			t = cast
		}
	} else {
		t = &Trace{}
	}

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
