package vparquet3

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/parquet-go/parquet-go"
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

func CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	s := newStreamingBlock(ctx, cfg, meta, r, to, tempo_io.NewBufferedWriter)

	var next func(context.Context) (common.ID, parquet.Row, error)

	if ii, ok := i.(*commonIterator); ok {
		// Use interal iterator and avoid translation to/from proto
		next = ii.NextRow
	} else {
		// Need to convert from proto->parquet obj
		trp := &Trace{}
		sch := parquet.SchemaOf(trp)
		next = func(context.Context) (common.ID, parquet.Row, error) {
			id, tr, err := i.Next(ctx)
			if errors.Is(err, io.EOF) || tr == nil {
				return id, nil, err
			}

			// Copy ID to allow it to escape the iterator.
			id = append([]byte(nil), id...)

			trp, _ = traceToParquet(meta, id, tr, trp) // this logic only executes when we are transitioning from one block version to another. just ignore connected here

			row := sch.Deconstruct(completeBlockRowPool.Get(), trp)

			return id, row, nil
		}
	}

	for {
		id, row, err := next(ctx)
		if errors.Is(err, io.EOF) || row == nil {
			break
		}

		err = s.AddRaw(id, row, 0, 0) // start and end time of the wal meta are used.
		if err != nil {
			return nil, err
		}
		completeBlockRowPool.Put(row)

		if s.EstimatedBufferedBytes() > cfg.RowGroupSizeBytes {
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
	pw    *parquet.GenericWriter[*Trace]
	w     *backendWriter
	r     backend.Reader
	to    backend.Writer
	index *index

	currentBufferedTraces int
	currentBufferedBytes  int
}

func newStreamingBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, r backend.Reader, to backend.Writer, createBufferedWriter func(w io.Writer) tempo_io.BufferedWriteFlusher) *streamingBlock {
	newMeta := backend.NewBlockMetaWithDedicatedColumns(meta.TenantID, meta.BlockID, VersionString, backend.EncNone, "", meta.DedicatedColumns)
	newMeta.StartTime = meta.StartTime
	newMeta.EndTime = meta.EndTime

	// TotalObjects is used here an an estimated count for the bloom filter.
	// The real number of objects is tracked below.
	bloom := common.NewBloom(cfg.BloomFP, uint(cfg.BloomShardSizeBytes), uint(meta.TotalObjects))

	w := &backendWriter{ctx, to, DataFileName, meta.BlockID, meta.TenantID, nil}
	bw := createBufferedWriter(w)
	pw := parquet.NewGenericWriter[*Trace](bw)

	return &streamingBlock{
		ctx:   ctx,
		meta:  newMeta,
		bloom: bloom,
		bw:    bw,
		pw:    pw,
		w:     w,
		r:     r,
		to:    to,
		index: &index{},
	}
}

func (b *streamingBlock) Add(tr *Trace, start, end uint32) error {
	_, err := b.pw.Write([]*Trace{tr})
	if err != nil {
		return err
	}
	id := tr.TraceID

	b.index.Add(id)
	b.bloom.Add(id)
	b.meta.ObjectAdded(id, start, end)
	b.currentBufferedTraces++
	b.currentBufferedBytes += estimateMarshalledSizeFromTrace(tr)

	return nil
}

func (b *streamingBlock) AddRaw(id []byte, row parquet.Row, start, end uint32) error {
	_, err := b.pw.WriteRows([]parquet.Row{row})
	if err != nil {
		return err
	}

	b.index.Add(id)
	b.bloom.Add(id)
	b.meta.ObjectAdded(id, start, end)
	b.currentBufferedTraces++
	b.currentBufferedBytes += estimateMarshalledSizeFromParquetRow(row)

	return nil
}

func (b *streamingBlock) EstimatedBufferedBytes() int {
	return b.currentBufferedBytes
}

func (b *streamingBlock) CurrentBufferedObjects() int {
	return b.currentBufferedTraces
}

func (b *streamingBlock) Flush() (int, error) {
	// Flush row group
	b.index.Flush()
	err := b.pw.Flush()
	if err != nil {
		return 0, err
	}

	n := b.bw.Len()
	b.meta.Size += uint64(n)
	b.meta.TotalRecords++
	b.currentBufferedTraces = 0
	b.currentBufferedBytes = 0

	// Flush to underlying writer
	return n, b.bw.Flush()
}

func (b *streamingBlock) Complete() (int, error) {
	// Flush final row group
	b.index.Flush()
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
	err = b.r.ReadRange(b.ctx, DataFileName, b.meta.BlockID, b.meta.TenantID, b.meta.Size-8, buf, nil)
	if err != nil {
		return 0, fmt.Errorf("error reading parquet file footer: %w", err)
	}
	if string(buf[4:8]) != "PAR1" {
		return 0, errors.New("Failed to confirm magic footer while writing a new parquet block")
	}
	b.meta.FooterSize = binary.LittleEndian.Uint32(buf[0:4])

	b.meta.BloomShardCount = uint16(b.bloom.GetShardCount())

	return n, writeBlockMeta(b.ctx, b.to, b.meta, b.bloom, b.index)
}

// estimateMarshalledSizeFromTrace attempts to estimate the size of trace in bytes. This is used to make choose
// when to cut a row group during block creation.
// TODO: This function regularly estimates lower values then estimateProtoSize() and the size
// of the actual proto. It's also quite inefficient. Perhaps just using static values per span or attribute
// would be a better choice?
func estimateMarshalledSizeFromTrace(tr *Trace) (size int) {
	size += 7 // 7 trace lvl fields

	for _, rs := range tr.ResourceSpans {
		size += estimateAttrSize(rs.Resource.Attrs)
		size += 10 // 10 resource span lvl fields

		for _, ils := range rs.ScopeSpans {
			size += 2 // 2 scope span lvl fields

			for _, s := range ils.Spans {
				size += 14 // 14 span lvl fields
				size += estimateAttrSize(s.Attrs)
				size += estimateEventsSize(s.Events)
			}
		}
	}
	return
}

func estimateAttrSize(attrs []Attribute) (size int) {
	return len(attrs) * 7 // 7 attribute lvl fields
}

func estimateEventsSize(events []Event) (size int) {
	for _, e := range events {
		size += 4                // 4 event lvl fields
		size += 4 * len(e.Attrs) // 2 event attribute fields
	}
	return
}
