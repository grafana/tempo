package vparquet

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"
	"github.com/willf/bloom"

	"github.com/grafana/tempo/pkg/parquetquery"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	SearchPrevious = -1
	SearchNext     = -2
	NotFound       = -3

	TraceIDColumnName = "TraceID"
)

type RowTracker struct {
	rgs         []parquet.RowGroup
	startRowNum []int

	// traceID column index
	colIndex int
}

// Scanning for a traceID within a rowGroup. Parameters are the rowgroup number and traceID to be searched.
// Includes logic to look through bloom filters and page bounds as it goes through the rowgroup.
func (rt *RowTracker) findTraceByID(idx int, traceID []byte) (int, error) {
	rgIdx := rt.rgs[idx]
	rowMatch := int64(rt.startRowNum[idx])
	traceIDColumnChunk := rgIdx.ColumnChunks()[rt.colIndex]

	bf := traceIDColumnChunk.BloomFilter()
	if bf != nil {
		exists, err := bf.Check(parquet.ValueOf(traceID))
		if err != nil {
			return NotFound, fmt.Errorf("error checking bloom filter: %w", err)
		}
		if !exists {
			return NotFound, nil
		}
	}

	// get row group bounds
	numPages := traceIDColumnChunk.ColumnIndex().NumPages()
	min := traceIDColumnChunk.ColumnIndex().MinValue(0).Bytes()
	max := traceIDColumnChunk.ColumnIndex().MaxValue(numPages - 1).Bytes()
	if bytes.Compare(traceID, min) < 0 {
		return SearchPrevious, nil
	}
	if bytes.Compare(max, traceID) < 0 {
		return SearchNext, nil
	}

	pages := traceIDColumnChunk.Pages()
	buffer := make([]parquet.Value, 10000)
	for {
		pg, err := pages.ReadPage()
		if pg == nil || err == io.EOF {
			break
		}

		if min, max, ok := pg.Bounds(); ok {
			if bytes.Compare(traceID, min.Bytes()) < 0 {
				return SearchPrevious, nil
			}
			if bytes.Compare(max.Bytes(), traceID) < 0 {
				rowMatch += pg.NumRows()
				continue
			}
		}

		vr := pg.Values()
		for {
			x, err := vr.ReadValues(buffer)
			for y := 0; y < x; y++ {
				if bytes.Equal(buffer[y].Bytes(), traceID) {
					rowMatch += int64(y)
					return int(rowMatch), nil
				}
			}

			// check for EOF after processing any returned data
			if err == io.EOF {
				break
			}
			if err != nil {
				return NotFound, err
			}

			rowMatch += int64(x)
		}
	}

	// did not find the trace
	return NotFound, nil
}

// Simple binary search algorithm over the parquet rowgroups to efficiently
// search for traceID in the block (works only because rows are sorted by traceID)
func (rt *RowTracker) binarySearch(span opentracing.Span, start int, end int, traceID []byte) (int, error) {
	if start > end {
		return -1, nil
	}

	// check mid point
	midResult, err := rt.findTraceByID((start+end)/2, traceID)
	if err != nil {
		return NotFound, err
	}
	span.LogFields(
		log.Message("checked mid result"),
		log.Int("start", start),
		log.Int("end", end),
		log.Int("midResult", midResult),
	)
	if midResult == SearchPrevious {
		return rt.binarySearch(span, start, ((start+end)/2)-1, traceID)
	} else if midResult < 0 {
		return rt.binarySearch(span, ((start+end)/2)+1, end, traceID)
	}

	return midResult, nil
}

func (b *backendBlock) checkBloom(ctx context.Context, id common.ID) (found bool, err error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.checkBloom",
		opentracing.Tags{
			"blockID":  b.meta.BlockID,
			"tenantID": b.meta.TenantID,
		})
	defer span.Finish()

	shardKey := common.ShardKeyForTraceID(id, int(b.meta.BloomShardCount))
	nameBloom := common.BloomName(shardKey)
	span.SetTag("bloom", nameBloom)

	bloomBytes, err := b.r.Read(derivedCtx, nameBloom, b.meta.BlockID, b.meta.TenantID, true)
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

func (b *backendBlock) FindTraceByID(ctx context.Context, traceID common.ID, opts common.SearchOptions) (_ *tempopb.Trace, err error) {
	return b.FindTraceByID2(ctx, traceID, opts)
}

func (b *backendBlock) FindTraceByID1(ctx context.Context, traceID common.ID, opts common.SearchOptions) (_ *tempopb.Trace, err error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.FindTraceByID",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	found, err := b.checkBloom(derivedCtx, traceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() {
		span.SetTag("inspectedBytes", rr.TotalBytesRead.Load())
		//fmt.Println("read bytes:", rr.TotalBytesRead.Load())
	}()

	// traceID column index
	colIndex, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
	if colIndex == -1 {
		return nil, fmt.Errorf("unable to get index for column: %s", TraceIDColumnName)
	}

	numRowGroups := len(pf.RowGroups())
	rt := &RowTracker{
		rgs:         make([]parquet.RowGroup, 0, numRowGroups),
		startRowNum: make([]int, 0, numRowGroups),

		colIndex: colIndex,
	}

	rowCount := 0
	for rgi := 0; rgi < numRowGroups; rgi++ {
		rt.rgs = append(rt.rgs, pf.RowGroups()[rgi])
		rt.startRowNum = append(rt.startRowNum, rowCount)
		rowCount += int(pf.RowGroups()[rgi].NumRows())
	}

	// find row number of matching traceID
	rowMatch, err := rt.binarySearch(span, 0, numRowGroups-1, traceID)
	if err != nil {
		return nil, errors.Wrap(err, "binary search")
	}

	// traceID not found in this block
	if rowMatch < 0 {
		return nil, nil
	}

	// seek to row and read
	r := parquet.NewReader(pf)
	err = r.SeekToRow(int64(rowMatch))
	if err != nil {
		return nil, errors.Wrap(err, "seek to row")
	}

	span.LogFields(log.Message("seeked to row"), log.Int("row", rowMatch))

	tr := new(Trace)
	err = r.Read(tr)
	if err != nil {
		return nil, errors.Wrap(err, "error reading row from backend")
	}

	span.LogFields(log.Message("read trace"))

	// convert to proto trace and return
	return parquetTraceToTempopbTrace(tr), nil
}

func (b *backendBlock) FindTraceByID2(ctx context.Context, traceID common.ID, opts common.SearchOptions) (_ *tempopb.Trace, err error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.FindTraceByID",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	found, err := b.checkBloom(derivedCtx, traceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	pf, rr, err := b.openForSearch(derivedCtx, opts, parquet.SkipPageIndex(true))
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() {
		span.SetTag("inspectedBytes", rr.TotalBytesRead.Load())
		//fmt.Println("read bytes:", rr.TotalBytesRead.Load())
	}()

	// traceID column index
	colIndex, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
	if colIndex == -1 {
		return nil, fmt.Errorf("unable to get index for column: %s", TraceIDColumnName)
	}

	numRowGroups := len(pf.RowGroups())

	// Cache of row group bounds
	rowGroupMins := make([]common.ID, numRowGroups)
	rowGroupMins[0] = b.meta.MinID
	rowGroupMaxs := make([]common.ID, numRowGroups)
	rowGroupMaxs[numRowGroups-1] = b.meta.MaxID // This is actually inclusive and the logic is special for the last row group below

	getRowGroupMin := func(rgIdx int) common.ID {
		min := rowGroupMins[rgIdx]
		if len(min) > 0 {
			// Already loaded
			return min
		}

		// Read first value from the row group
		rgs := pf.RowGroups()
		iter := parquetquery.NewColumnIterator(ctx, rgs[rgIdx:rgIdx+1], colIndex, "", 1, nil, "id")
		defer iter.Close()

		res, err := iter.Next()
		if err != nil {
			panic("failed to read 1 value for row group min")
		}
		if res == nil {
			panic("failed to read 1 value from row group")
		}

		min = res.ToMap()["id"][0].ByteArray()
		rowGroupMins[rgIdx] = min

		return min
	}

	getRowGroupMax := func(rgIdx int) common.ID {
		max := rowGroupMaxs[rgIdx]
		if len(max) > 0 {
			// Already loaded
			return max
		}

		// Read the min of the next row group is the best we can do
		max = getRowGroupMin(rgIdx + 1)
		rowGroupMaxs[rgIdx] = max
		return max
	}

	rowGroup := binarySearch(numRowGroups, func(rgIdx int) int {
		min := getRowGroupMin(rgIdx)
		if check := bytes.Compare(traceID, min); check <= 0 {
			// Trace is before or in this group
			return check
		}

		max := getRowGroupMax(rgIdx)
		// This is actually the min of the next group, so check is exclusive not inclusive like min
		// Except for the last group, it is inclusive
		check := bytes.Compare(traceID, max)
		if check > 0 || (check == 0 && rgIdx < (numRowGroups-1)) {
			// Trace is after this group
			return 1
		}

		// Must be in this group
		return 0
	})

	if rowGroup == -1 {
		// Not within the bounds of any row group
		return nil, nil
	}

	// Now iterate the matching row group
	iter := parquetquery.NewColumnIterator(ctx, pf.RowGroups()[rowGroup:rowGroup+1], colIndex, "", 1000, parquetquery.NewStringInPredicate([]string{string(traceID)}), "")
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
	rowMatch += res.RowNumber[0]

	// seek to row and read
	r := parquet.NewReader(pf)
	err = r.SeekToRow(int64(rowMatch))
	if err != nil {
		return nil, errors.Wrap(err, "seek to row")
	}

	span.LogFields(log.Message("seeked to row"), log.Int64("row", rowMatch))

	tr := new(Trace)
	err = r.Read(tr)
	if err != nil {
		return nil, errors.Wrap(err, "error reading row from backend")
	}

	span.LogFields(log.Message("read trace"))

	// convert to proto trace and return
	return parquetTraceToTempopbTrace(tr), nil
}

// binarySearch that finds exact matching entry. Returns non-zero index when found, or -1 when not found
// Inspired by sort.Search but makes uses of tri-state comparator to eliminate the last comparison when
// we want to find exact match, not insertion point.
func binarySearch(n int, compare func(int) int) int {
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		switch compare(h) {
		case 0:
			// Found exact match
			return h
		case -1:
			j = h
		case 1:
			i = h + 1
		}
	}

	// No match
	return -1
}

/*func dumpParquetRow(sch parquet.Schema, row parquet.Row) {
	for i, r := range row {
		slicestr := ""
		if r.Kind() == parquet.ByteArray {
			slicestr = util.TraceIDToHexString(r.ByteArray())
		}
		fmt.Printf("row[%d] = c:%d (%s) r:%d d:%d v:%s (%s)\n",
			i,
			r.Column(),
			strings.Join(sch.Columns()[r.Column()], "."),
			r.RepetitionLevel(),
			r.DefinitionLevel(),
			r.String(),
			slicestr,
		)
	}
}*/
