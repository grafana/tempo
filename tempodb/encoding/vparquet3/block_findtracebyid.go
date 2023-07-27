package vparquet3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"
	"github.com/willf/bloom"

	"github.com/grafana/tempo/pkg/parquetquery"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	SearchPrevious = -1
	SearchNext     = -2
	NotFound       = -3

	TraceIDColumnName = "TraceID"
)

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

func (b *backendBlock) checkIndex(ctx context.Context, id common.ID) (int, error) {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.checkIndex",
		opentracing.Tags{
			"blockID":  b.meta.BlockID,
			"tenantID": b.meta.TenantID,
		})
	defer span.Finish()

	indexBytes, err := b.r.Read(derivedCtx, common.NameIndex, b.meta.BlockID, b.meta.TenantID, true)
	if err != nil {
		return -1, fmt.Errorf("error retrieving index (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	index, err := unmarshalIndex(indexBytes)
	if err != nil {
		return -1, fmt.Errorf("error parsing index (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return index.Find(id), nil
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

	rowGroup, err := b.checkIndex(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if rowGroup < 0 {
		return nil, nil
	}

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() {
		span.SetTag("inspectedBytes", rr.BytesRead())
	}()

	return findTraceByID(derivedCtx, traceID, b.meta, pf, rowGroup)
}

func findTraceByID(ctx context.Context, traceID common.ID, meta *backend.BlockMeta, pf *parquet.File, rowGroup int) (*tempopb.Trace, error) {
	// traceID column index
	colIndex, _ := pq.GetColumnIndexByPath(pf, TraceIDColumnName)
	if colIndex == -1 {
		return nil, fmt.Errorf("unable to get index for column: %s", TraceIDColumnName)
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
	err = r.SeekToRow(rowMatch)
	if err != nil {
		return nil, errors.Wrap(err, "seek to row")
	}

	tr := new(Trace)
	err = r.Read(tr)
	if err != nil {
		return nil, errors.Wrap(err, "error reading row from backend")
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
