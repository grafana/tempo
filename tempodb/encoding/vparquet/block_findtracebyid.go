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
func (rt *RowTracker) findTraceByID(idx int, traceID []byte) int {
	rgIdx := rt.rgs[idx]
	rowMatch := int64(rt.startRowNum[idx])
	traceIDColumnChunk := rgIdx.ColumnChunks()[rt.colIndex]

	bf := traceIDColumnChunk.BloomFilter()
	if bf != nil {
		// todo: better error handling?
		exists, _ := bf.Check(parquet.ValueOf(traceID))
		if !exists {
			return NotFound
		}
	}

	// get row group bounds
	numPages := traceIDColumnChunk.ColumnIndex().NumPages()
	min := traceIDColumnChunk.ColumnIndex().MinValue(0).Bytes()
	max := traceIDColumnChunk.ColumnIndex().MaxValue(numPages - 1).Bytes()
	if bytes.Compare(traceID, min) < 0 {
		return SearchPrevious
	}
	if bytes.Compare(max, traceID) < 0 {
		return SearchNext
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
				return SearchPrevious
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
					return int(rowMatch)
				}
			}

			// check for EOF after processing any returned data
			if err == io.EOF {
				break
			}
			// todo: better error handling
			if err != nil {
				break
			}

			rowMatch += int64(x)
		}
	}

	// did not find the trace
	return NotFound
}

// Simple binary search algorithm over the parquet rowgroups to efficiently
// search for traceID in the block (works only because rows are sorted by traceID)
func (rt *RowTracker) binarySearch(span opentracing.Span, start int, end int, traceID []byte) int {
	if start > end {
		return -1
	}

	// check mid point
	midResult := rt.findTraceByID((start+end)/2, traceID)
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

	return midResult
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
		return false, fmt.Errorf("error retrieving bloom (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	filter := &bloom.BloomFilter{}
	_, err = filter.ReadFrom(bytes.NewReader(bloomBytes))
	if err != nil {
		return false, fmt.Errorf("error parsing bloom (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return filter.Test(id), nil
}

func (b *backendBlock) FindTraceByID(ctx context.Context, traceID common.ID, opts common.SearchOptions) (_ *tempopb.Trace, err error) {
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
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead.Load()) }()

	// traceID column index
	colIndex, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)

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
	rowMatch := rt.binarySearch(span, 0, numRowGroups-1, traceID)

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
	return parquetTraceToTempopbTrace(tr)
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
