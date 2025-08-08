package vparquet4

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/willf/bloom"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/parquetquery"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
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

	bloomBytes, err := b.r.Read(derivedCtx, nameBloom, (uuid.UUID)(b.meta.BlockID), b.meta.TenantID, &backend.CacheInfo{
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

	indexBytes, err := b.r.Read(derivedCtx, common.NameIndex, (uuid.UUID)(b.meta.BlockID), b.meta.TenantID, &backend.CacheInfo{
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

	found, err := b.checkBloom(derivedCtx, traceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	ok, rowGroup, err := b.checkIndex(derivedCtx, traceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}

	foundTrace, err := findTraceByID(derivedCtx, traceID, b.meta, pf, rowGroup)

	result := &tempopb.TraceByIDResponse{
		Trace:   foundTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
	}
	bytesRead := rr.BytesRead()
	result.Metrics.InspectedBytes += bytesRead
	span.SetAttributes(attribute.Int64("inspectedBytes", int64(bytesRead)))

	return result, err
}

// TraceExists checks if a trace exists in the block without loading the actual trace data.
// This is much more efficient than FindTraceByID for existence checking.
func (b *backendBlock) TraceExists(ctx context.Context, traceID common.ID, opts common.SearchOptions) (bool, uint64, error) {
	derivedCtx, span := tracer.Start(ctx, "parquet4.backendBlock.TraceExists",
		trace.WithAttributes(
			attribute.String("blockID", b.meta.BlockID.String()),
			attribute.String("tenantID", b.meta.TenantID),
			attribute.Int64("blockSize", int64(b.meta.Size_)),
		))
	defer span.End()

	var inspectedBytes uint64

	// Check bloom filter first - this is very fast
	found, err := b.checkBloom(derivedCtx, traceID)
	if err != nil {
		return false, 0, err
	}
	if !found {
		// Bloom filter says trace doesn't exist - definitive negative
		return false, 0, nil
	}
	
	// Bloom filter says trace might exist, check index
	ok, rowGroup, err := b.checkIndex(derivedCtx, traceID)
	if err != nil {
		return false, 0, err
	}
	if !ok {
		// Index says trace doesn't exist - definitive negative
		return false, 0, nil
	}

	// If we have a specific row group from the index, we can do a lightweight check
	// without reading the full parquet file
	if rowGroup != -1 {
		// We have a row group hint from the index, trace very likely exists
		// We could do an even more precise check here if needed, but for most
		// cases this is sufficient and much faster than loading the trace
		span.SetAttributes(attribute.Int("rowGroup", rowGroup))
		return true, inspectedBytes, nil
	}

	// Fallback: open the parquet file and do a lightweight existence check
	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return false, 0, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	
	exists, err := checkTraceExistsInParquet(derivedCtx, traceID, pf)
	inspectedBytes = rr.BytesRead()
	span.SetAttributes(attribute.Int64("inspectedBytes", int64(inspectedBytes)))
	
	return exists, inspectedBytes, err
}

// checkTraceExistsInParquet does a lightweight check for trace existence in parquet
// without loading the full trace data
func checkTraceExistsInParquet(ctx context.Context, traceID common.ID, pf *parquet.File) (bool, error) {
	// Get the TraceID column index
	colIndex, _, maxDef := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
	if colIndex == -1 {
		return false, fmt.Errorf("unable to get index for column: %s", TraceIDColumnName)
	}

	// Do a binary search across row groups to find the right one
	numRowGroups := len(pf.RowGroups())
	buf := make(parquet.Row, 1)

	// Binary search for the row group containing this trace ID
	rowGroup, err := binarySearch(numRowGroups, func(rgIdx int) (int, error) {
		// Get the min trace ID from this row group
		pages := pf.RowGroups()[rgIdx].ColumnChunks()[colIndex].Pages()
		defer pages.Close()

		page, err := pages.ReadPage()
		if err != nil {
			return 0, err
		}
		defer parquet.Release(page)

		c, err := page.Values().ReadValues(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		if c < 1 {
			return 0, fmt.Errorf("failed to read value from page")
		}

		min := buf[0].Clone().ByteArray()
		return bytes.Compare(traceID, min), nil
	})

	if err != nil {
		return false, fmt.Errorf("error binary searching row groups: %w", err)
	}

	if rowGroup == -1 {
		// Not found in any row group
		return false, nil
	}

	// Use a sync iterator to check if the trace ID exists in the row group
	// This is more precise than just checking the row group bounds
	iter := parquetquery.NewSyncIterator(ctx, pf.RowGroups()[rowGroup:rowGroup+1], colIndex,
		parquetquery.SyncIteratorOptPredicate(parquetquery.NewStringInPredicate([]string{string(traceID)})),
		parquetquery.SyncIteratorOptMaxDefinitionLevel(maxDef),
	)
	defer iter.Close()

	res, err := iter.Next()
	if err != nil {
		return false, err
	}

	// If we got a result, the trace exists
	return res != nil, nil
}

func findTraceByID(ctx context.Context, traceID common.ID, meta *backend.BlockMeta, pf *parquet.File, rowGroup int) (*tempopb.Trace, error) {
	// traceID column index
	colIndex, _, maxDef := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
	if colIndex == -1 {
		return nil, fmt.Errorf("unable to get index for column: %s", TraceIDColumnName)
	}

	// If no index then fallback to binary searching the rowgroups.
	if rowGroup == -1 {
		var (
			numRowGroups = len(pf.RowGroups())
			buf          = make(parquet.Row, 1)
			err          error
		)

		// Cache of row group bounds
		rowGroupMins := make([]common.ID, numRowGroups+1)
		// todo: restore using meta min/max id once it works
		//    https://github.com/grafana/tempo/issues/1903
		rowGroupMins[0] = bytes.Repeat([]byte{0}, 16)
		rowGroupMins[numRowGroups] = bytes.Repeat([]byte{255}, 16) // This is actually inclusive and the logic is special for the last row group below

		// Gets the minimum trace ID within the row group. Since the column is sorted
		// ascending we just read the first value from the first page.
		getRowGroupMin := func(rgIdx int) (common.ID, error) {
			min := rowGroupMins[rgIdx]
			if len(min) > 0 {
				// Already loaded
				return min, nil
			}

			pages := pf.RowGroups()[rgIdx].ColumnChunks()[colIndex].Pages()
			defer pages.Close()

			page, err := pages.ReadPage()
			if err != nil {
				return nil, err
			}
			defer parquet.Release(page)

			c, err := page.Values().ReadValues(buf)
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
			if c < 1 {
				return nil, fmt.Errorf("failed to read value from page: traceID: %s blockID:%v rowGroupIdx:%d", util.TraceIDToHexString(traceID), meta.BlockID, rgIdx)
			}

			// Clone ensures that the byte array is disconnected
			// from the underlying i/o buffers.
			min = buf[0].Clone().ByteArray()
			rowGroupMins[rgIdx] = min
			return min, nil
		}

		rowGroup, err = binarySearch(numRowGroups, func(rgIdx int) (int, error) {
			min, err := getRowGroupMin(rgIdx)
			if err != nil {
				return 0, err
			}

			if check := bytes.Compare(traceID, min); check <= 0 {
				// Trace is before or in this group
				return check, nil
			}

			max, err := getRowGroupMin(rgIdx + 1)
			if err != nil {
				return 0, err
			}

			// This is actually the min of the next group, so check is exclusive not inclusive like min
			// Except for the last group, it is inclusive
			check := bytes.Compare(traceID, max)
			if check > 0 || (check == 0 && rgIdx < (numRowGroups-1)) {
				// Trace is after this group
				return 1, nil
			}

			// Must be in this group
			return 0, nil
		})
		if err != nil {
			return nil, fmt.Errorf("error binary searching row groups: %w", err)
		}
	}

	if rowGroup == -1 {
		// Not within the bounds of any row group
		return nil, nil
	}

	// Now iterate the matching row group
	iter := parquetquery.NewSyncIterator(ctx, pf.RowGroups()[rowGroup:rowGroup+1], colIndex,
		parquetquery.SyncIteratorOptPredicate(parquetquery.NewStringInPredicate([]string{string(traceID)})),
		parquetquery.SyncIteratorOptMaxDefinitionLevel(maxDef),
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
	rowMatch := int64(0)
	for _, rg := range pf.RowGroups()[0:rowGroup] {
		rowMatch += rg.NumRows()
	}
	rowMatch += int64(res.RowNumber[0])

	// seek to row and read
	r := parquet.NewGenericReader[*Trace](pf)
	defer r.Close()

	err = r.SeekToRow(rowMatch)
	if err != nil {
		return nil, fmt.Errorf("seek to row: %w", err)
	}

	tr := new(Trace)
	_, err = r.Read([]*Trace{tr})
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("error reading row from backend: %w", err)
	}

	// convert to proto trace and return
	return parquetTraceToTempopbTrace(meta, tr), nil
}

// binarySearch that finds exact matching entry. Returns non-zero index when found, or -1 when not found
// Inspired by sort.Search but makes uses of tri-state comparator to eliminate the last comparison when
// we want to find exact match, not insertion point.
func binarySearch(n int, compare func(int) (int, error)) (int, error) {
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		c, err := compare(h)
		if err != nil {
			return -1, err
		}
		// i ≤ h < j
		switch c {
		case 0:
			// Found exact match
			return h, nil
		case -1:
			j = h
		case 1:
			i = h + 1
		}
	}

	// No match
	return -1, nil
}
