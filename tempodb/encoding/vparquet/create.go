package vparquet

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/google/uuid"
	tempo_io "github.com/grafana/tempo/pkg/io"
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

func CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	s := newStreamingBlock(ctx, cfg, meta, r, to, tempo_io.NewBufferedWriter)

	trp := &Trace{}
	for {
		id, tr, err := i.Next(ctx)
		if err == io.EOF || tr == nil {
			break
		}

		// Copy ID to allow it to escape the iterator.
		id = append([]byte(nil), id...)

		trp = traceToParquet(id, tr, trp)
		err = s.Add(trp, 0, 0) // start and end time of the wal meta are used.
		if err != nil {
			return nil, err
		}

		// Here we repurpose RowGroupSizeBytes as number of raw column values.
		// This is a fairly close approximation.
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

	currentBufferedTraces int
	currentBufferedBytes  int
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
	}
}

func (b *streamingBlock) Add(tr *Trace, start, end uint32) error {
	_, err := b.pw.Write([]*Trace{tr})
	if err != nil {
		return err
	}
	id := tr.TraceID

	b.bloom.Add(id)
	b.meta.ObjectAdded(id, start, end)
	b.currentBufferedTraces++
	b.currentBufferedBytes += estimateTraceSize(tr)

	return nil
}

func (b *streamingBlock) AddRaw(id []byte, row parquet.Row, start, end uint32) error {
	_, err := b.pw.WriteRows([]parquet.Row{row})
	if err != nil {
		return err
	}

	b.bloom.Add(id)
	b.meta.ObjectAdded(id, start, end)
	b.currentBufferedTraces++
	b.currentBufferedBytes += estimateProtoSize(row)

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
	err = b.r.ReadRange(b.ctx, DataFileName, b.meta.BlockID, b.meta.TenantID, b.meta.Size-8, buf, false)
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

// estimateTraceSize attempts to estimate the size of trace in bytes. This is used to make choose
// when to cut a row group during block creation.
// TODO: This function regularly estimates lower values then estimateProtoSize() and the size
// of the actual proto. It's also quite inefficient. Perhaps just using static values per span or attribute
// would be a better choice?
func estimateTraceSize(tr *Trace) (size int) {
	size += len(tr.TraceID)
	size += len(tr.TraceIDText)
	size += len(tr.RootServiceName)
	size += len(tr.RootSpanName)
	size += 8 + 8 + 8 // start/end/duration
	size += 7

	for _, rs := range tr.ResourceSpans {
		size += estimateAttrSize(rs.Resource.Attrs)
		size += len(rs.Resource.ServiceName)
		size += strLen(rs.Resource.Namespace)
		size += strLen(rs.Resource.Cluster)
		size += strLen(rs.Resource.Pod)
		size += strLen(rs.Resource.Container)
		size += strLen(rs.Resource.K8sClusterName)
		size += strLen(rs.Resource.K8sContainerName)
		size += strLen(rs.Resource.K8sNamespaceName)
		size += strLen(rs.Resource.K8sPodName)
		size += 9

		for _, ils := range rs.ScopeSpans {
			size += len(ils.Scope.Name)
			size += len(ils.Scope.Version)
			size += 2
			for _, s := range ils.Spans {
				size += 8 + 8 + 8 + 8 + 4 + 4 // start/end/kind/statuscode/dropped events/dropped attrs
				size += len(s.ID)
				size += len(s.ParentSpanID)
				size += len(s.Name)
				size += strLen(s.HttpMethod)
				size += strLen(s.HttpUrl)
				size += len(s.StatusMessage)
				size += len(s.TraceState)
				if s.HttpStatusCode != nil {
					size += 8
				}
				size += estimateAttrSize(s.Attrs)
				size += estimateEventsSize(s.Events)
				size += 14
			}
		}
	}
	return
}

func estimateAttrSize(attrs []Attribute) (size int) {
	for _, a := range attrs {
		size += len(a.Key)
		size += strLen(a.Value)
		size += len(a.ValueArray)
		size += len(a.ValueKVList)
		if a.ValueBool != nil {
			size++
		}
		if a.ValueDouble != nil {
			size += 8
		}
		if a.ValueInt != nil {
			size += 8
		}
	}
	return
}

func estimateEventsSize(events []Event) (size int) {
	for _, e := range events {
		size += 8 + 4 // time/dropped attributes
		size += len(e.Name)

		for _, eva := range e.Attrs {
			size += len(eva.Value)
			size += len(eva.Key)
		}
	}
	return
}

func strLen(s *string) (size int) {
	if s == nil {
		return 0
	}
	return len(*s)
}
