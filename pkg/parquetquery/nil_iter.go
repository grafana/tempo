package parquetquery

import (
	"context"
	"errors"
	"fmt"
	"io"

	pq "github.com/parquet-go/parquet-go"
)

// NilSyncIterator copies all functions of the sync iterator with just the next() function being different
type NilSyncIterator struct {
	SyncIterator
	lastRowNumberReturned RowNumber
	valueFound            bool
}

var _ Iterator = (*NilSyncIterator)(nil)

func NewNilSyncIterator(ctx context.Context, rgs []pq.RowGroup, column int, opts ...SyncIteratorOpt) *NilSyncIterator {
	// Create the sync iterator
	syncIterator := NewSyncIterator(ctx, rgs, column, opts...)

	i := &NilSyncIterator{
		SyncIterator:          *syncIterator,
		lastRowNumberReturned: EmptyRowNumber(),
		valueFound:            false,
	}

	return i
}

func (c *NilSyncIterator) String() string {
	filter := "nil"
	if c.filter != nil {
		filter = c.filter.String()
	}
	return fmt.Sprintf("NilSyncIterator: %s : %s", c.columnName, filter)
}

func (c *NilSyncIterator) Next() (*IteratorResult, error) {
	rn, _, err := c.next()
	if err != nil {
		return nil, err
	}

	if !rn.Valid() {
		return nil, nil
	}

	nilValue := pq.ValueOf(nil)
	v := &nilValue
	return c.makeResult(rn, v), nil
}

func (c *NilSyncIterator) next() (RowNumber, *pq.Value, error) {
	var lastValue *pq.Value
	lastRowNumber := EmptyRowNumber()
	for {
		if c.currRowGroup == nil {
			rg, minRN, maxRN := c.popRowGroup()
			if rg == nil {
				// no more rows, return last row if we still haven't found the value
				if lastValue != nil && !c.valueFound && lastRowNumber.Valid() && !EqualRowNumber(lastValue.DefinitionLevel(), c.lastRowNumberReturned, lastRowNumber) {
					c.lastRowNumberReturned = lastRowNumber
					return lastRowNumber, lastValue, nil
				}
				return EmptyRowNumber(), nil, nil
			}

			cc := &ColumnChunkHelper{ColumnChunk: rg.ColumnChunks()[c.column]}
			if c.filter != nil && !c.filter.KeepColumnChunk(cc) {
				cc.Close()
				continue
			}

			c.setRowGroup(rg, minRN, maxRN, cc)
		}

		if c.currPage == nil {
			pg, err := c.currChunk.NextPage()
			if pg == nil || errors.Is(err, io.EOF) {
				// This row group is exhausted
				c.closeCurrRowGroup()
				continue
			}
			if err != nil {
				return EmptyRowNumber(), nil, err
			}
			if c.filter != nil && !c.filter.KeepPage(pg) {
				// This page filtered out
				c.curr.Skip(pg.NumRows())
				pq.Release(pg)
				continue
			}
			c.setPage(pg)
		}

		// Read next batch of values if needed
		if c.currBuf == nil {
			c.currBuf = syncIteratorPoolGet(c.readSize, 0)
		}

		if c.currBufN >= len(c.currBuf) || len(c.currBuf) == 0 {
			c.currBuf = c.currBuf[:cap(c.currBuf)]
			n, err := c.currValues.ReadValues(c.currBuf)
			if err != nil && !errors.Is(err, io.EOF) {
				return EmptyRowNumber(), nil, err
			}
			c.currBuf = c.currBuf[:n]
			c.currBufN = 0
			if n == 0 {
				// This value reader and page are exhausted.
				c.setPage(nil)
				continue
			}
		}

		// Consume current buffer until empty
		for c.currBufN < len(c.currBuf) {
			v := &c.currBuf[c.currBufN]

			if v.RepetitionLevel() < v.DefinitionLevel() && !v.IsNull() {
				// moving on to the next level higher than value level
				// so if we haven't seen the value yet, it does not exist
				// check if we've already returned this row so we can properly next()
				if !c.valueFound && lastRowNumber.Valid() && !EqualRowNumber(c.maxDefinitionLevel, c.lastRowNumberReturned, c.curr) {
					c.lastRowNumberReturned = lastRowNumber
					return lastRowNumber, lastValue, nil
				}
				// new level reset
				c.valueFound = false
			}

			// Inspect all values to track the current row number,
			// even if the value is filtered out next.
			c.curr.Next(v.RepetitionLevel(), v.DefinitionLevel(), c.maxDefinitionLevel)
			c.currBufN++
			c.currPageN++

			if v.IsNull() {
				continue
			}

			if c.filter != nil && c.filter.KeepValue(*v) {
				c.valueFound = true
			}

			lastValue = v
			lastRowNumber = c.curr
			continue
		}
	}
}
