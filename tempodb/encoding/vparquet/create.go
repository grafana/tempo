package vparquet

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/util"
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
	flushSize := 30_000_000

	s, err := NewStreamingBlock(ctx, cfg, meta, to)
	if err != nil {
		return nil, err
	}

	for {
		_, obj, err := i.Next(ctx)
		if err == io.EOF {
			break
		}

		tr, err := dec.PrepareForRead(obj)
		if err != nil {
			return nil, err
		}

		trp := traceToParquet(tr)
		err = s.Add(&trp)
		if err != nil {
			return nil, err
		}

		if s.CurrentBufferLength() > flushSize {
			_, err = s.Flush()
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = s.Complete()
	if err != nil {
		return nil, err
	}

	return s.meta, nil
}

type streamingBlock struct {
	ctx   context.Context
	bloom *common.ShardedBloomFilter
	meta  *backend.BlockMeta
	bw    *tempo_io.BufferedWriter
	pw    *parquet.Writer
	w     *backendWriter
	to    backend.Writer

	currentBufferedTraces int
}

func NewStreamingBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, to backend.Writer) (*streamingBlock, error) {
	newMeta := backend.NewBlockMeta(meta.TenantID, meta.BlockID, VersionString, backend.EncNone, "")
	newMeta.StartTime = meta.StartTime
	newMeta.EndTime = meta.EndTime

	// TotalObjects is used here an an estimated count for the bloom filter.
	// The real number of objects is tracked below.
	bloom := common.NewBloom(cfg.BloomFP, uint(cfg.BloomShardSizeBytes), uint(meta.TotalObjects))

	w := &backendWriter{ctx, to, "data.parquet", meta.BlockID, meta.TenantID, nil}

	bw := tempo_io.NewBufferedWriter(w)

	sch := parquet.SchemaOf(new(Trace))

	pw := parquet.NewWriter(bw, sch, &parquet.WriterConfig{PageBufferSize: 10_000_000})

	return &streamingBlock{
		ctx:   ctx,
		meta:  newMeta,
		bloom: bloom,
		bw:    bw,
		pw:    pw,
		w:     w,
		to:    to,
	}, nil
}

func (b *streamingBlock) Add(tr *Trace) error {

	err := b.pw.Write(tr)
	if err != nil {
		return err
	}

	id, err := util.HexStringToTraceID(tr.TraceID)
	if err != nil {
		return err
	}

	b.bloom.Add(id)
	start := uint32(tr.StartTimeUnixNano / uint64(time.Second))
	b.meta.ObjectAdded(id, start, start+uint32(tr.DurationNanos/uint64(time.Second)))
	b.currentBufferedTraces++
	return nil
}

func (b *streamingBlock) CurrentBufferLength() int {
	return b.bw.Len()
}

func (b *streamingBlock) CurrentBufferedObjects() int {
	return b.currentBufferedTraces
}

func (b *streamingBlock) Flush() (int, error) {
	// Flush row group
	err := b.pw.Flush()
	if err != nil {
		return 0, err
	}

	n := b.bw.Len()
	b.meta.Size += uint64(n)
	b.meta.TotalRecords++

	// Flush to underlying writer
	return n, b.bw.Flush()
}

func (b *streamingBlock) Complete() (int, error) {
	// Flush final row group
	b.meta.TotalRecords++
	err := b.pw.Flush()
	if err != nil {
		return 0, err
	}

	// Close parquet file. This writes the footer and metadata.
	err = b.pw.Close()
	if err != nil {
		return 0, err
	}

	// Now Flush and close out in-memory buffer
	n := b.bw.Len()
	b.meta.Size += uint64(n)
	err = b.bw.Flush()
	if err != nil {
		return 0, err
	}

	err = b.bw.Close()
	if err != nil {
		return 0, err
	}

	err = b.w.Close()
	if err != nil {
		return 0, err
	}

	b.meta.BloomShardCount = uint16(b.bloom.GetShardCount())

	return n, writeBlockMeta(b.ctx, b.to, b.meta, b.bloom)
}
