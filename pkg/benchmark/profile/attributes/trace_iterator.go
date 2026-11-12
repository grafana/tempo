package attributes

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"

	tempo_io "github.com/grafana/tempo/v3/pkg/io"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet5"
)

// The iterator reads every column of every row group once, in order, so reads
// are buffered in large chunks as analyse block does.
const (
	traceReadBufferSize  = 2 * 1024 * 1024
	traceReadBufferCount = 64
)

// openTraceIterator reads a block's traces as OTLP. Profiling from traces
// rather than columns keeps it independent of a version's layout: another
// version only has to produce its traces here.
func openTraceIterator(ctx context.Context, meta *backend.BlockMeta, r backend.Reader) (common.Iterator, error) {
	switch meta.Version {
	case vparquet5.VersionString:
		return newVParquet5TraceIterator(ctx, meta, r)
	default:
		return nil, fmt.Errorf("reading traces from %s blocks is not supported", meta.Version)
	}
}

type vparquet5TraceIterator struct {
	meta *backend.BlockMeta
	r    *parquet.GenericReader[*vparquet5.Trace]
}

func newVParquet5TraceIterator(ctx context.Context, meta *backend.BlockMeta, r backend.Reader) (*vparquet5TraceIterator, error) {
	size := int64(meta.Size_)
	ra := tempo_io.NewBufferedReaderAt(vparquet5.NewBackendReaderAt(ctx, r, vparquet5.DataFileName, meta), size, traceReadBufferSize, traceReadBufferCount)

	// Dedicated columns can change the schema, such as blobs written without a
	// dictionary, so the block is read with its own as the block iterator does.
	sch, _, readerOptions := vparquet5.SchemaWithDynamicChanges(meta.DedicatedColumns)
	pf, err := parquet.OpenFile(ra, size, parquet.SkipBloomFilters(true), parquet.SkipPageIndex(true), parquet.FileSchema(sch))
	if err != nil {
		return nil, fmt.Errorf("opening %s for block %s: %w", vparquet5.DataFileName, meta.BlockID, err)
	}
	gr := parquet.NewGenericReader[*vparquet5.Trace](pf, append(readerOptions, sch)...)
	return &vparquet5TraceIterator{meta: meta, r: gr}, nil
}

// Next returns io.EOF after the last trace.
func (i *vparquet5TraceIterator) Next(context.Context) (common.ID, *tempopb.Trace, error) {
	traces := []*vparquet5.Trace{{}}
	// The last trace can arrive together with io.EOF.
	n, err := i.r.Read(traces)
	if n == 0 {
		if err == nil || errors.Is(err, io.EOF) {
			return nil, nil, io.EOF
		}
		return nil, nil, err
	}

	tr := traces[0]
	return tr.TraceID, vparquet5.ParquetTraceToTempopbTrace(i.meta, tr), nil
}

func (i *vparquet5TraceIterator) Close() {
	_ = i.r.Close()
}
