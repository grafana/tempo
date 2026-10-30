package vparquet5

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	"github.com/grafana/tempo/v3/pkg/cache"
	pq "github.com/grafana/tempo/v3/pkg/parquetquery"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

const (
	SearchPrevious = -1
	SearchNext     = -2
	NotFound       = -3

	TraceIDColumnName = "TraceID"

	EnvVarIndexName         = "VPARQUET_INDEX"
	EnvVarIndexEnabledValue = "1"
)

func (b *backendBlock) checkBloom(ctx context.Context, id common.ID) (found bool, err error) {
	derivedCtx, span := tracer.Start(ctx, "parquet.backendBlock.checkBloom",
		trace.WithAttributes(
			attribute.String("blockID", b.meta.BlockID.String()),
			attribute.String("tenantID", b.meta.TenantID),
		))
	defer span.End()

	shardKey := common.ShardKeyForTraceID(id, int(b.meta.BloomShardCount))
	nameBloom := common.BloomName(shardKey)
	span.SetAttributes(attribute.String("bloom", nameBloom))

	bloomBytes, err := b.r.Read(derivedCtx, nameBloom, uuid.UUID(b.meta.BlockID), b.meta.TenantID, &backend.CacheInfo{
		Meta: b.meta,
		Role: cache.RoleBloom,
	})
	if err != nil {
		return false, fmt.Errorf("error retrieving bloom %s (%s, %s): %w", nameBloom, b.meta.TenantID, b.meta.BlockID, err)
	}

	filter := &bloom.BloomFilter{}
	_, err = filter.ReadFrom(bytes.NewReader(bloomBytes))
	if err != nil {
		return false, fmt.Errorf("error parsing bloom (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return filter.Test(id), nil
}

func (b *backendBlock) checkIndex(ctx context.Context, id common.ID) (bool, int, error) {
	if os.Getenv(EnvVarIndexName) != EnvVarIndexEnabledValue {
		// Index lookup disabled
		return true, -1, nil
	}

	derivedCtx, span := tracer.Start(ctx, "parquet4.backendBlock.checkIndex",
		trace.WithAttributes(
			attribute.String("blockID", b.meta.BlockID.String()),
			attribute.String("tenantID", b.meta.TenantID),
		))
	defer span.End()

	indexBytes, err := b.r.Read(derivedCtx, common.NameIndex, uuid.UUID(b.meta.BlockID), b.meta.TenantID, &backend.CacheInfo{
		Meta: b.meta,
		Role: cache.RoleTraceIDIdx,
	})
	if errors.Is(err, backend.ErrDoesNotExist) {
		return true, -1, nil
	}
	if err != nil {
		return false, -1, fmt.Errorf("error retrieving index (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	index, err := unmarshalIndex(indexBytes)
	if err != nil {
		return false, -1, fmt.Errorf("error parsing index (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	rowGroup := index.Find(id)
	if rowGroup == -1 {
		// Ruled out by index
		return false, -1, nil
	}

	return true, rowGroup, nil
}

func (b *backendBlock) FindTraceByID(ctx context.Context, traceID common.ID, opts common.SearchOptions) (_ *tempopb.TraceByIDResponse, err error) {
	derivedCtx, span := tracer.Start(ctx, "parquet.backendBlock.FindTraceByID",
		trace.WithAttributes(
			attribute.String("blockID", b.meta.BlockID.String()),
			attribute.String("tenantID", b.meta.TenantID),
			attribute.Int64("blockSize", int64(b.meta.Size_)),
		))
	defer span.End()

	common.MetricFindBlocks.WithLabelValues(b.meta.TenantID, common.FindBlocksStageBloomChecked).Inc()
	found, err := b.checkBloom(derivedCtx, traceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	common.MetricFindBlocks.WithLabelValues(b.meta.TenantID, common.FindBlocksStageBloomPass).Inc()

	ok, rowGroup, err := b.checkIndex(derivedCtx, traceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	common.MetricFindBlocks.WithLabelValues(b.meta.TenantID, common.FindBlocksStageIndexPass).Inc()

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	common.MetricFindBlocks.WithLabelValues(b.meta.TenantID, common.FindBlocksStageRead).Inc()

	foundTrace, err := findTraceByID(derivedCtx, traceID, b.meta, pf, rr, rowGroup)

	result := &tempopb.TraceByIDResponse{
		Trace:   foundTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
	}
	bytesRead := rr.BytesRead()
	result.Metrics.InspectedBytes += bytesRead
	span.SetAttributes(attribute.Int64("inspectedBytes", int64(bytesRead)))

	return result, err
}

func findTraceByID(ctx context.Context, traceID common.ID, meta *backend.BlockMeta, pf *parquet.File, rr *BackendReaderAt, rowGroup int) (*tempopb.Trace, error) {
	// traceID column index
	colIndex, _, maxDef := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
	if colIndex == -1 {
		return nil, fmt.Errorf("unable to get index for column: %s", TraceIDColumnName)
	}

	rowGroups := pf.RowGroups()

	if rowGroup == -1 {
		var ok bool
		rowGroup, ok = findRowGroupByStats(pf, colIndex, traceID)
		if ok && rowGroup == -1 {
			return nil, nil
		}
	}

	rowMatch := int64(-1)

	if rowGroup >= 0 {
		// Load the column index of the trace ID column and the offset indexes
		// of the row group, so the page containing the trace ID is located via
		// the indexes instead of scanning the row group.
		if err := preloadPageIndexes(rr, pf, rowGroup, colIndex); err != nil {
			return nil, fmt.Errorf("error loading page indexes: %w", err)
		}

		row, ok, err := pq.FindRowByColumnIndex(rowGroups[rowGroup], colIndex, traceID)
		if err != nil {
			return nil, fmt.Errorf("error locating trace ID by column index: %w", err)
		}
		if ok {
			if row == -1 {
				// Ruled out by the column index: the trace is not in this block.
				return nil, nil
			}
			// The row number is relative to the row group, so offset it using
			// the num rows in all previous groups.
			rowMatch = row
			for _, rg := range pf.RowGroups()[:rowGroup] {
				rowMatch += rg.NumRows()
			}
		}
	}

	if rowMatch == -1 {
		// The row group is unknown (statistics missing, all row groups scanned)
		// or its column index can't be used: scan for the trace ID instead.
		firstRowGroup := 0
		if rowGroup != -1 {
			firstRowGroup = rowGroup
			rowGroups = rowGroups[rowGroup : rowGroup+1]
		}

		iter := pq.NewSyncIterator(
			ctx, rowGroups, colIndex,
			pq.SyncIteratorOptPredicate(pq.NewStringInPredicate([]string{string(traceID)})),
			pq.SyncIteratorOptMaxDefinitionLevel(maxDef),
		)
		defer iter.Close()

		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if res == nil {
			// TraceID not found in this block
			return nil, nil
		}

		// The row number coming out of the iterator is relative,
		// so offset it using the num rows in all previous groups
		rowMatch = 0
		for _, rg := range pf.RowGroups()[0:firstRowGroup] {
			rowMatch += rg.NumRows()
		}
		rowMatch += int64(res.RowNumber[0])
	}

	// seek to row and read
	_, _, readerOptions := SchemaWithDynamicChanges(meta.DedicatedColumns)
	r := parquet.NewGenericReader[*Trace](pf, readerOptions...)
	defer r.Close()

	err := r.SeekToRow(rowMatch)
	if err != nil {
		return nil, fmt.Errorf("seek to row: %w", err)
	}

	tr := new(Trace)
	_, err = r.Read([]*Trace{tr})
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("error reading row from backend: %w", err)
	}

	// convert to proto trace and return
	return ParquetTraceToTempopbTrace(meta, tr), nil
}

// findRowGroupByStats returns the row group whose footer min/max contains traceID, or -1.
// ok is false if any row group lacks statistics.
func findRowGroupByStats(pf *parquet.File, colIndex int, traceID common.ID) (rowGroup int, ok bool) {
	rgs := pf.Metadata().RowGroups
	for _, rg := range rgs {
		s := rg.Columns[colIndex].MetaData.Statistics
		if len(s.MinValue) == 0 || len(s.MaxValue) == 0 {
			return -1, false
		}
	}

	i := sort.Search(len(rgs), func(i int) bool {
		return bytes.Compare(rgs[i].Columns[colIndex].MetaData.Statistics.MaxValue, traceID) >= 0
	})
	if i == len(rgs) || bytes.Compare(traceID, rgs[i].Columns[colIndex].MetaData.Statistics.MinValue) < 0 {
		return -1, true
	}
	return i, true
}

// preloadPageIndexes loads the column index of the given column and the offset
// index of every column chunk in the row group.
// With the column index, the page containing a value is found by binary search
// on the page min/max bounds, instead of scanning the row group. With the
// offset indexes, SeekToRow jumps directly to the page containing a row
// instead of reading and decoding all pages from the start of the row group.
// A row group's indexes are written contiguously at the end of the file, so
// each is read in a single request, and the two regions are read concurrently.
// Column chunks without an index are skipped: SeekToRow reads sequentially for
// them, and pq.FindRowByColumnIndex falls back to scanning the row group.
func preloadPageIndexes(r *BackendReaderAt, pf *parquet.File, rowGroup, colIndex int) error {
	cols := pf.Metadata().RowGroups[rowGroup].Columns

	var g errgroup.Group

	// The column index of the searched column.
	if offset, length := cols[colIndex].ColumnIndexOffset, int64(cols[colIndex].ColumnIndexLength); offset != 0 {
		g.Go(func() error {
			buf := make([]byte, length)
			if _, err := r.ReadAtWithCache(buf, offset, cache.RoleParquetColumnIdx); err != nil {
				return err
			}
			cc := pf.RowGroups()[rowGroup].ColumnChunks()[colIndex]
			if fcc, ok := cc.(*parquet.FileColumnChunk); ok {
				_, err := fcc.ColumnIndexFrom(&regionReaderAt{base: offset, data: buf})
				if err != nil && !errors.Is(err, parquet.ErrMissingColumnIndex) {
					return err
				}
			}
			return nil
		})
	}

	// The offset index of every column chunk.
	base, end := int64(-1), int64(0)
	for i := range cols {
		offset, length := cols[i].OffsetIndexOffset, cols[i].OffsetIndexLength
		if offset == 0 {
			continue
		}
		if base == -1 || offset < base {
			base = offset
		}
		if e := offset + int64(length); e > end {
			end = e
		}
	}
	if base != -1 {
		g.Go(func() error {
			buf := make([]byte, end-base)
			if _, err := r.ReadAtWithCache(buf, base, cache.RoleParquetOffsetIdx); err != nil {
				return err
			}

			region := &regionReaderAt{base: base, data: buf}
			for _, cc := range pf.RowGroups()[rowGroup].ColumnChunks() {
				fcc, ok := cc.(*parquet.FileColumnChunk)
				if !ok {
					continue
				}
				if _, err := fcc.OffsetIndexFrom(region); err != nil && !errors.Is(err, parquet.ErrMissingOffsetIndex) {
					return err
				}
			}
			return nil
		})
	}

	return g.Wait()
}

// regionReaderAt serves ReadAt calls at absolute file offsets from an
// in-memory copy of a file region starting at base.
type regionReaderAt struct {
	base int64
	data []byte
}

func (r *regionReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < r.base {
		return 0, io.EOF
	}
	n := copy(p, r.data[off-r.base:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
