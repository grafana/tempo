package vparquet

import (
	"context"
	"io"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/segmentio/parquet-go"
)

func (b *backendBlock) Iterator(ctx context.Context) (*iterator, error) {
	rr := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)

	// 16 MB memory buffering
	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), 512*1024, 32)

	pf, err := parquet.OpenFile(br, int64(b.meta.Size))
	if err != nil {
		return nil, err
	}

	r := parquet.NewReader(pf, parquet.SchemaOf(&Trace{}))

	return &iterator{r}, nil
}

type iterator struct {
	r *parquet.Reader
}

func (i *iterator) Next() (*Trace, error) {
	t := &Trace{}
	switch err := i.r.Read(t); err {
	case nil:
		return t, nil
	case io.EOF:
		return nil, nil
	default:
		return nil, err
	}
}
