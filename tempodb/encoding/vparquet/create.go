package vparquet

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
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

func CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, dec model.ObjectDecoder, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	s := newStreamingBlock(ctx, cfg, meta, r, to, tempo_io.NewBufferedWriter)

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
		err = s.Add(&trp, 0, 0) // start and end time of the wal meta are used.
		if err != nil {
			return nil, err
		}

		if s.CurrentBufferLength() > cfg.RowGroupSizeBytes || s.CurrentBufferedObjects() > 10_000 {
			_, err = s.Flush()
			if err != nil {
				return nil, err
			}
		}
	}

	_, err := s.Complete()
	if err != nil {
		return nil, err
	}

	return s.meta, nil
}

type streamingBlock struct {
	ctx   context.Context
	bloom *common.ShardedBloomFilter
	meta  *backend.BlockMeta
	bw    tempo_io.BufferedWriteFlusher
	pw    *parquet.Writer
	w     *backendWriter
	r     backend.Reader
	to    backend.Writer

	currentBufferedTraces int
}

func newStreamingBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, r backend.Reader, to backend.Writer, createBufferedWriter func(w io.Writer) tempo_io.BufferedWriteFlusher) *streamingBlock {
	newMeta := backend.NewBlockMeta(meta.TenantID, meta.BlockID, VersionString, backend.EncNone, "")
	newMeta.StartTime = meta.StartTime
	newMeta.EndTime = meta.EndTime

	// TotalObjects is used here an an estimated count for the bloom filter.
	// The real number of objects is tracked below.
	bloom := common.NewBloom(cfg.BloomFP, uint(cfg.BloomShardSizeBytes), uint(meta.TotalObjects))

	w := &backendWriter{ctx, to, DataFileName, meta.BlockID, meta.TenantID, nil}

	bw := createBufferedWriter(w)

	sch := parquet.SchemaOf(new(Trace))

	// Since we store the trace ID as 32-byte hex string, we have to increase the
	// column/page index size so that min/max return the whole value instead of
	// a truncated form. This allows FindTraceByID to use binary search accurately.
	pw := parquet.NewWriter(bw, sch, parquet.ColumnIndexSizeLimit(32))

	return &streamingBlock{
		ctx:   ctx,
		meta:  newMeta,
		bloom: bloom,
		bw:    bw,
		pw:    pw,
		w:     w,
		r:     r,
		to:    to,
	}
}

func (b *streamingBlock) Add(tr *Trace, start, end uint32) error {

	err := b.pw.Write(tr)
	if err != nil {
		return err
	}

	id, err := util.HexStringToTraceID(tr.TraceID)
	if err != nil {
		return err
	}

	b.bloom.Add(id)
	b.meta.ObjectAdded(id, start, end)
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
	b.currentBufferedTraces = 0

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

	// Read the footer size out of the parquet footer
	buf := make([]byte, 8)
	err = b.r.ReadRange(b.ctx, DataFileName, b.meta.BlockID, b.meta.TenantID, b.meta.Size-8, buf)
	if err != nil {
		return 0, errors.Wrap(err, "error reading parquet file footer")
	}
	if string(buf[4:8]) != "PAR1" {
		return 0, errors.New("Failed to confirm magic footer while writing a new parquet block")
	}
	b.meta.FooterSize = binary.LittleEndian.Uint32(buf[0:4])

	b.meta.BloomShardCount = uint16(b.bloom.GetShardCount())

	return n, writeBlockMeta(b.ctx, b.to, b.meta, b.bloom)
}
