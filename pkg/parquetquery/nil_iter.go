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
}

var _ Iterator = (*NilSyncIterator)(nil)

func NewNilSyncIterator(ctx context.Context, rgs []pq.RowGroup, column int, opts ...SyncIteratorOpt) *NilSyncIterator {
	// Create the sync iterator
	syncIterator := NewSyncIterator(ctx, rgs, column, opts...)

	i := &NilSyncIterator{
		SyncIterator:          *syncIterator,
		lastRowNumberReturned: EmptyRowNumber(),
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

func (c *NilSyncIterator) SeekTo(to RowNumber, definitionLevel int) (*IteratorResult, error) {
	for {
		if done := c.seekRowGroup(to, definitionLevel); done {
			return nil, nil
		}

		done, err := c.seekPages(to, definitionLevel)
		if err != nil {
			return nil, err
		}
		if done {
			// This row group is exhausted try the next one.
			continue
		}

		c.seekWithinPage(to, definitionLevel)
		break
	}

	// The row group and page have been selected to where this value is possibly
	// located. Now scan through the page and look for it.
	for {
		rn, v, err := c.next()
		if err != nil {
			return nil, err
		}
		if !rn.Valid() {
			return nil, nil
		}

		if CompareRowNumbers(definitionLevel, rn, to) >= 0 {
			return c.makeResult(rn, v), nil
		}
	}
}

func (c *NilSyncIterator) next() (RowNumber, *pq.Value, error) {
	var (
		scopeRow       = EmptyRowNumber()
		scopeHasValues bool
		valueFound     bool
		emptyNilValue  pq.Value
	)

	// This is called right before we exit a scope of repeated values.
	// We emit a nil response if we got at least one value and never saw the filter match.
	tryEmitNilOnScopeExit := func() (RowNumber, bool) {
		if valueFound || !scopeHasValues || !scopeRow.Valid() {
			return RowNumber{}, false
		}
		if EqualRowNumber(c.maxDefinitionLevel, c.lastRowNumberReturned, c.curr) {
			return RowNumber{}, false
		}
		c.lastRowNumberReturned = scopeRow
		return scopeRow, true
	}

	advanceValue := func(v *pq.Value) {
		c.curr.Next(v.RepetitionLevel(), v.DefinitionLevel(), c.maxDefinitionLevel)
		c.currBufN++
		c.currPageN++

		if !v.IsNull() {
			scopeHasValues = true
		}
		if c.filter != nil && c.filter.KeepValue(*v) {
			valueFound = true
		}
	}

	for {
		if c.currRowGroup == nil {
			rg, minRN, maxRN := c.popRowGroup()
			if rg == nil {
				// No more rows, maybe return the last one if it matches criteria.
				if rn, ok := tryEmitNilOnScopeExit(); ok {
					return rn, &emptyNilValue, nil
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
			if err != nil && !errors.Is(err, io.EOF) {
				return EmptyRowNumber(), nil, err
			}
			if pg == nil || errors.Is(err, io.EOF) {
				// This row group is exhausted
				c.closeCurrRowGroup()
				continue
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
			var (
				v    = &c.currBuf[c.currBufN]
				r    = v.RepetitionLevel()
				d    = v.DefinitionLevel()
				maxD = c.maxDefinitionLevel
			)

			if r < maxD {
				// This means we are moving on to the next row.
				// Before doing so, see if we need to emit a response for the row we are exiting.
				if rn, ok := tryEmitNilOnScopeExit(); ok {
					return rn, &emptyNilValue, nil
				}

				// new level reset
				valueFound = false
				scopeHasValues = false
				advanceValue(v)
				scopeRow = c.curr

				if r <= d && d == maxD-1 && v.IsNull() {
					// Empty repeated values for this level, which means the value doesn't exist.
					// However because we checking that we are the second to last level, it means
					// we are also ensuring there is an owning row defined at this level.
					// In this case we emit a nil immediately upon entering the scope
					// because this is the only row for it.
					c.lastRowNumberReturned = c.curr
					return c.curr, &emptyNilValue, nil
				}

				// Neither of the above cases matched,
				// so we are just entering a new row
				continue
			}

			// Inspect all values to track the current row number,
			// even if the value is filtered out next.
			advanceValue(v)
		}
	}
}
