package parquetquery

import (
	"context"
	"errors"
	"fmt"
	"io"

	pq "github.com/parquet-go/parquet-go"
)

// NilAttributeIterator copies all functions of the sync iterator with just the next() function being different
type NilAttributeIterator struct {
	syncIterator          SyncIterator
	lastRowNumberReturned RowNumber
	attrFound             bool
	lastBuff              bool
}

var _ Iterator = (*NilAttributeIterator)(nil)

func NewNilAttributeIterator(ctx context.Context, rgs []pq.RowGroup, column int, columnName string, readSize int, filter Predicate, selectAs string, maxDefinitionLevel int, opts ...SyncIteratorOpt) *NilAttributeIterator {
	// Create the sync iterator
	syncIterator := NewSyncIterator(ctx, rgs, column, opts...)
	// Apply options
	for _, opt := range opts {
		opt(syncIterator)
	}

	i := &NilAttributeIterator{
		syncIterator:          *syncIterator,
		lastRowNumberReturned: EmptyRowNumber(),
		attrFound:             false,
		lastBuff:              false,
	}

	return i
}

func (c *NilAttributeIterator) String() string {
	filter := "nil"
	if c.syncIterator.filter != nil {
		filter = c.syncIterator.filter.String()
	}
	return fmt.Sprintf("NilAttributeIterator: %s : %s", c.syncIterator.columnName, filter)
}

func (c *NilAttributeIterator) Next() (*IteratorResult, error) {
	rn, v, err := c.next()
	if err != nil {
		return nil, err
	}
	if !rn.Valid() {
		return nil, nil
	}
	return c.makeResult(rn, v), nil
}

func (c *NilAttributeIterator) SeekTo(to RowNumber, definitionLevel int) (*IteratorResult, error) {
	return c.syncIterator.SeekTo(to, definitionLevel)
}

func (c *NilAttributeIterator) popRowGroup() (pq.RowGroup, RowNumber, RowNumber) {
	return c.syncIterator.popRowGroup()
}

func (c *NilAttributeIterator) seekRowGroup(seekTo RowNumber, definitionLevel int) (done bool) {
	return c.syncIterator.seekRowGroup(seekTo, definitionLevel)
}

func (c *NilAttributeIterator) seekPages(seekTo RowNumber, definitionLevel int) (done bool, err error) {
	return c.syncIterator.seekPages(seekTo, definitionLevel)
}

func (c *NilAttributeIterator) seekWithinPage(to RowNumber, definitionLevel int) {
	c.syncIterator.seekWithinPage(to, definitionLevel)
}

func (c *NilAttributeIterator) next() (RowNumber, *pq.Value, error) {
	for {
		if c.syncIterator.currRowGroup == nil {
			rg, min, max := c.popRowGroup()
			if rg == nil {
				return EmptyRowNumber(), nil, nil
			}

			cc := &ColumnChunkHelper{ColumnChunk: rg.ColumnChunks()[c.syncIterator.column]}
			if c.syncIterator.filter != nil && !c.syncIterator.filter.KeepColumnChunk(cc) {
				cc.Close()
				continue
			}

			c.setRowGroup(rg, min, max, cc)
		}

		if c.syncIterator.currPage == nil {
			pg, err := c.syncIterator.currChunk.NextPage()
			if pg == nil || errors.Is(err, io.EOF) {
				// This row group is exhausted
				c.closeCurrRowGroup()
				continue
			}
			if err != nil {
				return EmptyRowNumber(), nil, err
			}
			if c.syncIterator.filter != nil && !c.syncIterator.filter.KeepPage(pg) {
				// This page filtered out
				c.syncIterator.curr.Skip(pg.NumRows())
				pq.Release(pg)
				continue
			}
			c.setPage(pg)
		}

		// Read next batch of values if needed
		if c.syncIterator.currBuf == nil {
			c.syncIterator.currBuf = syncIteratorPoolGet(c.syncIterator.readSize, 0)
		}

		if c.syncIterator.currBufN >= len(c.syncIterator.currBuf) || len(c.syncIterator.currBuf) == 0 {
			c.syncIterator.currBuf = c.syncIterator.currBuf[:cap(c.syncIterator.currBuf)]
			n, err := c.syncIterator.currValues.ReadValues(c.syncIterator.currBuf)
			if err != nil && !errors.Is(err, io.EOF) {
				return EmptyRowNumber(), nil, err
			}
			if errors.Is(err, io.EOF) {
				c.lastBuff = true
			}
			c.syncIterator.currBuf = c.syncIterator.currBuf[:n]
			c.syncIterator.currBufN = 0
			if n == 0 {
				// This value reader and page are exhausted.
				c.setPage(nil)
				continue
			}
		}

		// Consume current buffer until empty
		for c.syncIterator.currBufN < len(c.syncIterator.currBuf) {
			v := &c.syncIterator.currBuf[c.syncIterator.currBufN]

			if v.RepetitionLevel() < v.DefinitionLevel() {
				// new level reset
				c.attrFound = false
			}

			// Inspect all values to track the current row number,
			// even if the value is filtered out next.
			c.syncIterator.curr.Next(v.RepetitionLevel(), v.DefinitionLevel(), c.syncIterator.maxDefinitionLevel)
			c.syncIterator.currBufN++
			c.syncIterator.currPageN++

			if c.syncIterator.filter != nil && c.syncIterator.filter.KeepValue(*v) {
				c.attrFound = true
				continue
			}

			// if this is the last value then check here
			if c.lastBuff && c.syncIterator.currBufN == len(c.syncIterator.currBuf) && !c.attrFound && c.syncIterator.curr.Valid() && !EqualRowNumber(v.DefinitionLevel(), c.lastRowNumberReturned, c.syncIterator.curr) {
				c.lastRowNumberReturned = c.syncIterator.curr
				return c.syncIterator.curr, v, nil
			}

			// if not last value then check if the next value puts us at a higher level
			if c.syncIterator.currBufN < len(c.syncIterator.currBuf) {
				nextV := &c.syncIterator.currBuf[c.syncIterator.currBufN]
				if nextV.RepetitionLevel() < nextV.DefinitionLevel() {
					// moving on to the next level higher than attribute level
					// so if we haven't seen the attribute yet, it is nil
					// check if we've already returned this row so we can properly next()
					if !c.attrFound && c.syncIterator.curr.Valid() && !EqualRowNumber(v.DefinitionLevel(), c.lastRowNumberReturned, c.syncIterator.curr) {
						c.lastRowNumberReturned = c.syncIterator.curr
						return c.syncIterator.curr, v, nil
					}

				}
			}

			continue
		}
	}
}

func (c *NilAttributeIterator) setRowGroup(rg pq.RowGroup, min, max RowNumber, cc *ColumnChunkHelper) {
	c.syncIterator.setRowGroup(rg, min, max, cc)
}

func (c *NilAttributeIterator) setPage(pg pq.Page) {
	c.syncIterator.setPage(pg)
}

func (c *NilAttributeIterator) closeCurrRowGroup() {
	c.syncIterator.closeCurrRowGroup()
}

func (c *NilAttributeIterator) makeResult(t RowNumber, v *pq.Value) *IteratorResult {
	return c.syncIterator.makeResult(t, v)
}

func (c *NilAttributeIterator) Close() {
	c.syncIterator.Close()
}
