package vparquet4

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/grafana/tempo/pkg/parquetquery"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/parquet-go/parquet-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracesCheck checks if a trace exists in the block without loading the actual trace data.
// This is much more efficient than FindTraceByID for existence checking.
func (b *backendBlock) TracesCheck(ctx context.Context, traceID common.ID, opts common.SearchOptions) (bool, uint64, error) {
	derivedCtx, span := tracer.Start(ctx, "parquet4.backendBlock.TracesCheck",
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