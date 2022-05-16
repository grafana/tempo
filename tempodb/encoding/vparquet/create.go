package vparquet

import (
	"context"
	"io"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/segmentio/parquet-go"
)

type backendWriter struct {
	ctx      context.Context
	w        backend.Writer
	name     string
	blockID  uuid.UUID
	tenantID string
	tracker  backend.AppendTracker
}

var _ io.WriteCloser = (*backendWriter)(nil)

func (b *backendWriter) Write(p []byte) (n int, err error) {
	b.tracker, err = b.w.Append(b.ctx, b.name, b.blockID, b.tenantID, b.tracker, p)
	return len(p), err
}

func (b *backendWriter) Close() error {
	return b.w.CloseAppend(b.ctx, b.tracker)
}

func CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, dec model.ObjectDecoder, to backend.Writer) (*backend.BlockMeta, error) {
	newMeta := backend.NewBlockMeta(meta.TenantID, meta.BlockID, VersionString, backend.EncNone, "")
	newMeta.StartTime = meta.StartTime
	newMeta.EndTime = meta.EndTime

	flushSize := 30_000_000

	bloom := common.NewBloom(cfg.BloomFP, uint(cfg.BloomShardSizeBytes), uint(meta.TotalObjects))

	ww := &backendWriter{ctx, to, "data.parquet", meta.BlockID, meta.TenantID, nil}

	bw := tempo_io.NewBufferedWriter(ww)

	sch := parquet.SchemaOf(new(Trace))

	w := parquet.NewWriter(ww, sch, &parquet.WriterConfig{PageBufferSize: 10_000_000})

	for {

		id, obj, err := i.Next(ctx)
		if err == io.EOF {
			break
		}

		tr, err := dec.PrepareForRead(obj)
		if err != nil {
			return nil, err
		}

		trp := traceToParquet(tr)

		err = w.Write(trp)
		if err != nil {
			return nil, err
		}

		bloom.Add(id)
		meta.TotalObjects++

		if bw.Len() > flushSize {
			// Flush row group
			err = w.Flush()
			if err != nil {
				return nil, err
			}

			newMeta.Size += uint64(bw.Len())
			newMeta.TotalRecords++

			err = bw.Flush()
			if err != nil {
				return nil, err
			}
		}
	}

	// Flush final row group and end of file
	newMeta.TotalRecords++
	err := w.Flush()
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	// Flush and close out buffer too
	newMeta.Size += uint64(bw.Len())
	err = bw.Flush()
	if err != nil {
		return nil, err
	}

	err = bw.Close()
	if err != nil {
		return nil, err
	}

	err = ww.Close()
	if err != nil {
		return nil, err
	}

	newMeta.BloomShardCount = uint16(bloom.GetShardCount())

	err = writeBlockMeta(ctx, to, newMeta, bloom)
	if err != nil {
		return nil, err
	}

	return newMeta, nil
}
