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

// TracesCheck checks if multiple traces exist in the block without loading the actual trace data.
// This is much more efficient than calling TracesCheck individually for each trace.
func (b *backendBlock) TracesCheck(ctx context.Context, traceIDs []common.ID, opts common.SearchOptions) (map[string]bool, uint64, error) {
	derivedCtx, span := tracer.Start(ctx, "parquet4.backendBlock.TracesCheck",
		trace.WithAttributes(
			attribute.String("blockID", b.meta.BlockID.String()),
			attribute.String("tenantID", b.meta.TenantID),
			attribute.Int64("blockSize", int64(b.meta.Size_)),
			attribute.Int("traceCount", len(traceIDs)),
		))
	defer span.End()

	results := make(map[string]bool, len(traceIDs))
	var totalInspectedBytes uint64
	
	// Initialize all results as false
	for _, traceID := range traceIDs {
		results[string(traceID)] = false
	}

	// First pass: check bloom filter for all trace IDs
	// This can quickly eliminate many traces that definitely don't exist
	candidateIDs := make([]common.ID, 0, len(traceIDs))
	for _, traceID := range traceIDs {
		found, err := b.checkBloom(derivedCtx, traceID)
		if err != nil {
			return nil, 0, err
		}
		if found {
			// Bloom filter says trace might exist
			candidateIDs = append(candidateIDs, traceID)
		}
		// If bloom filter says no, results[traceID] stays false
	}

	if len(candidateIDs) == 0 {
		// Bloom filter eliminated all traces
		return results, 0, nil
	}

	// Second pass: check index for remaining candidates
	indexCandidates := make([]common.ID, 0, len(candidateIDs))
	for _, traceID := range candidateIDs {
		ok, rowGroup, err := b.checkIndex(derivedCtx, traceID)
		if err != nil {
			return nil, 0, err
		}
		if ok {
			if rowGroup != -1 {
				// Index gives us a specific row group, very likely exists
				results[string(traceID)] = true
			} else {
				// Index says it might exist but no specific row group
				indexCandidates = append(indexCandidates, traceID)
			}
		}
		// If index says no, results[string(traceID)] stays false
	}

	if len(indexCandidates) == 0 {
		// Index resolved all remaining candidates
		return results, totalInspectedBytes, nil
	}

	// Third pass: check parquet file for remaining candidates
	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	
	// Check all remaining candidates in the parquet file
	for _, traceID := range indexCandidates {
		exists, err := checkTraceExistsInParquet(derivedCtx, traceID, pf)
		if err != nil {
			return nil, 0, err
		}
		results[string(traceID)] = exists
	}
	
	totalInspectedBytes = rr.BytesRead()
	span.SetAttributes(attribute.Int64("totalInspectedBytes", int64(totalInspectedBytes)))
	
	return results, totalInspectedBytes, nil
}