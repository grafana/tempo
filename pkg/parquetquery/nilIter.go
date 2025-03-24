package parquetquery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/grafana/tempo/pkg/parquetquery/intern"

	pq "github.com/parquet-go/parquet-go"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type NilAttributeIteratorOpt func(*NilAttributeIterator)

// NilAttributeIteratorOptIntern enables interning of string values.
// This is useful when the same string value is repeated many times.
// Not recommended with (very) high cardinality columns, such as UUIDs (spanID and traceID).
func NilAttributeIteratorOptIntern() NilAttributeIteratorOpt {
	return func(i *NilAttributeIterator) {
		i.intern = true
		i.interner = intern.New()
	}
}

// NilAttributeIterator is like ColumnIterator but synchronous. It scans through the given row
// groups and column, and applies the optional predicate to each chunk, page, and value.
// Results are read by calling Next() until it returns nil.
type NilAttributeIterator struct {
	// Config
	column     int
	columnName string
	rgs        []pq.RowGroup
	rgsMin     []RowNumber
	rgsMax     []RowNumber // Exclusive, row number of next one past the row group
	readSize   int
	filter     Predicate

	// Status
	span                     trace.Span
	curr                     RowNumber
	currRowGroup             pq.RowGroup
	currRowGroupMin          RowNumber
	currRowGroupMax          RowNumber
	currChunk                *ColumnChunkHelper
	currPage                 pq.Page
	currPageMin              RowNumber
	currPageMax              RowNumber
	currValues               pq.ValueReader
	currBuf                  []pq.Value
	currBufN                 int
	currPageN                int
	at                       IteratorResult // Current value pointed at by iterator. Returned by call Next and SeekTo, valid until next call.
	lastNonMatchingRowNumber RowNumber

	intern   bool
	interner *intern.Interner
}

var _ Iterator = (*NilAttributeIterator)(nil)

func NewNilAttributeIterator(ctx context.Context, rgs []pq.RowGroup, column int, columnName string, readSize int, filter Predicate, selectAs string, opts ...NilAttributeIteratorOpt) *NilAttributeIterator {
	// Assign row group bounds.
	// Lower bound is inclusive
	// Upper bound is exclusive, points at the first row of the next group
	rn := EmptyRowNumber()
	rgsMin := make([]RowNumber, len(rgs))
	rgsMax := make([]RowNumber, len(rgs))
	for i, rg := range rgs {
		rgsMin[i] = rn
		rgsMax[i] = rn
		rgsMax[i].Skip(rg.NumRows() + 1)
		rn.Skip(rg.NumRows())
	}

	_, span := tracer.Start(ctx, "NilAttributeIterator", trace.WithAttributes(
		attribute.Int("columnIndex", column),
		attribute.String("column", columnName),
	))

	at := IteratorResult{}
	if selectAs != "" {
		// Preallocate 1 entry with the given name.
		at.Entries = []struct {
			Key   string
			Value pq.Value
		}{
			{Key: selectAs},
		}
	}

	// Create the iterator
	i := &NilAttributeIterator{
		span:       span,
		column:     column,
		columnName: columnName,
		rgs:        rgs,
		readSize:   readSize,
		rgsMin:     rgsMin,
		rgsMax:     rgsMax,
		filter:     filter,
		curr:       EmptyRowNumber(),
		at:         at,
	}

	// Apply options
	for _, opt := range opts {
		opt(i)
	}

	return i
}

func (c *NilAttributeIterator) String() string {
	filter := "nil"
	if c.filter != nil {
		filter = c.filter.String()
	}
	return fmt.Sprintf("NilAttributeIterator: %s : %s", c.columnName, filter)
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

// SeekTo moves this iterator to the next result that is greater than
// or equal to the given row number (and based on the given definition level)
func (c *NilAttributeIterator) SeekTo(to RowNumber, definitionLevel int) (*IteratorResult, error) {
	if c.seekRowGroup(to, definitionLevel) {
		return nil, nil
	}

	done, err := c.seekPages(to, definitionLevel)
	if err != nil {
		return nil, err
	}
	if done {
		return nil, nil
	}

	c.seekWithinPage(to, definitionLevel)

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

func (c *NilAttributeIterator) popRowGroup() (pq.RowGroup, RowNumber, RowNumber) {
	if len(c.rgs) == 0 {
		return nil, EmptyRowNumber(), EmptyRowNumber()
	}

	rg := c.rgs[0]
	min := c.rgsMin[0]
	max := c.rgsMax[0]

	c.rgs = c.rgs[1:]
	c.rgsMin = c.rgsMin[1:]
	c.rgsMax = c.rgsMax[1:]

	return rg, min, max
}

// seekRowGroup skips ahead to the row group that could contain the value at the
// desired row number. Does nothing if the current row group is already the correct one.
func (c *NilAttributeIterator) seekRowGroup(seekTo RowNumber, definitionLevel int) (done bool) {
	if c.currRowGroup != nil && CompareRowNumbers(definitionLevel, seekTo, c.currRowGroupMax) >= 0 {
		// Done with this row group
		c.closeCurrRowGroup()
	}

	for c.currRowGroup == nil {

		rg, min, max := c.popRowGroup()
		if rg == nil {
			return true
		}

		if CompareRowNumbers(definitionLevel, seekTo, max) != -1 {
			continue
		}

		cc := &ColumnChunkHelper{ColumnChunk: rg.ColumnChunks()[c.column]}
		if c.filter != nil && !c.filter.KeepColumnChunk(cc) {
			cc.Close()
			continue
		}

		// This row group matches both row number and filter.
		c.setRowGroup(rg, min, max, cc)
	}

	return c.currRowGroup == nil
}

// seekPages skips ahead in the current row group to the page that could contain the value at
// the desired row number. Does nothing if the current page is already the correct one.
func (c *NilAttributeIterator) seekPages(seekTo RowNumber, definitionLevel int) (done bool, err error) {
	if c.currPage != nil && CompareRowNumbers(definitionLevel, seekTo, c.currPageMax) >= 0 {
		// Value not in this page
		c.setPage(nil)
	}

	if c.currPage == nil {
		// TODO (mdisibio)   :((((((((
		//    pages.SeekToRow is more costly than expected.  It doesn't reuse existing i/o
		// so it can't be called naively every time we swap pages. We need to figure out
		// a way to determine when it is worth calling here.
		/*
			// Seek into the pages. This is relative to the start of the row group
			if seekTo[0] > 0 {
				// Determine row delta. We subtract 1 because curr points at the previous row
				skip := seekTo[0] - c.currRowGroupMin[0] - 1
				if skip > 0 {
					if err := c.currPages.SeekToRow(skip); err != nil {
						return true, err
					}
					c.curr.Skip(skip)
				}
			}*/

		for c.currPage == nil {
			pg, err := c.currChunk.NextPage()
			if pg == nil || err != nil {
				// No more pages in this column chunk,
				// cleanup and exit.
				if errors.Is(err, io.EOF) {
					err = nil
				}
				pq.Release(pg)
				c.closeCurrRowGroup()
				return true, err
			}

			// Skip based on row number?
			newRN := c.curr
			newRN.Skip(pg.NumRows() + 1)
			if CompareRowNumbers(definitionLevel, seekTo, newRN) >= 0 {
				c.curr.Skip(pg.NumRows())
				pq.Release(pg)
				continue
			}

			// Skip based on filter?
			if c.filter != nil && !c.filter.KeepPage(pg) {
				c.curr.Skip(pg.NumRows())
				pq.Release(pg)
				continue
			}

			c.setPage(pg)
		}
	}

	return false, nil
}

// seekWithinPage decides if it should reslice the current page to jump directly to the desired row number
// or allow the iterator to call Next() until it finds the desired row number. it uses the magicThreshold
// as its balance point. if the number of Next()s to skip is less than the magicThreshold, it will not reslice
func (c *NilAttributeIterator) seekWithinPage(to RowNumber, definitionLevel int) {
	rowSkipRelative := int(to[0] - c.curr[0])
	if rowSkipRelative == 0 {
		return
	}

	const magicThreshold = 1000
	shouldSkip := false

	if definitionLevel == 0 {
		// if definition level is 0 there is always a 1:1 ratio between Next()s and rows. it's only deeper
		// levels of nesting we have to manually count
		shouldSkip = rowSkipRelative > magicThreshold
	} else {
		// this is a nested iterator, let's count the Next()s required to get to the desired row number
		// and decide if we should skip or not
		replvls := c.currPage.RepetitionLevels()
		nextsRequired := 0

		for i := c.currPageN; i < len(replvls); i++ {
			nextsRequired++

			if nextsRequired > magicThreshold {
				shouldSkip = true
				break
			}

			if replvls[i] == 0 { // 0 rep lvl indicates a new row
				rowSkipRelative-- // decrement the number of rows we need to skip
				if rowSkipRelative <= 0 {
					// if we hit here we skipped all rows and did not exceed the magic threshold, so we're leaving shouldSkip false
					break
				}
			}
		}
	}

	if !shouldSkip {
		return
	}

	// skips are calculated off the start of the page
	rowSkip := to[0] - c.currPageMin[0]
	if rowSkip < 1 {
		return
	}
	if rowSkip > int32(c.currPage.NumRows()) {
		return
	}

	// reslice the page to jump directly to the desired row number
	pg := c.currPage.Slice(int64(rowSkip-1), c.currPage.NumRows())

	// remove all detail below the row number
	c.curr = TruncateRowNumber(0, to)
	c.curr = c.curr.Preceding()

	// reset buffers and other vars
	pq.Release(c.currPage)
	c.currPage = pg
	c.currPageMin = c.curr
	c.currValues = pg.Values()
	c.currPageN = 0
	nilAttributeIteratorPoolPut(c.currBuf)
	c.currBuf = nil
}

// next is the core functionality of this iterator and returns the next matching result. This
// may involve inspecting multiple row groups, pages, and values until a match is found. When
// we run out of things to inspect, it returns nil. The reason this method is distinct from
// Next() is because it doesn't wrap the results in an IteratorResult, which is more efficient
// when being called multiple times and throwing away the results like in SeekTo().
func (c *NilAttributeIterator) next() (RowNumber, *pq.Value, error) {
	attrFound := false
	for {
		if c.currRowGroup == nil {
			rg, min, max := c.popRowGroup()
			if rg == nil {
				return EmptyRowNumber(), nil, nil
			}

			cc := &ColumnChunkHelper{ColumnChunk: rg.ColumnChunks()[c.column]}
			if c.filter != nil && !c.filter.KeepColumnChunk(cc) {
				cc.Close()
				continue
			}

			c.setRowGroup(rg, min, max, cc)
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
			c.currBuf = nilAttributeIteratorPoolGet(c.readSize, 0)
		}

		lastBuff := false
		if c.currBufN >= len(c.currBuf) || len(c.currBuf) == 0 {
			c.currBuf = c.currBuf[:cap(c.currBuf)]
			n, err := c.currValues.ReadValues(c.currBuf)
			if err != nil && !errors.Is(err, io.EOF) {
				return EmptyRowNumber(), nil, err
			}
			if errors.Is(err, io.EOF) {
				lastBuff = true
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

			// Inspect all values to track the current row number,
			// even if the value is filtered out next.
			c.curr.Next(v.RepetitionLevel(), v.DefinitionLevel())
			c.currBufN++
			c.currPageN++

			fmt.Printf("nil iterator rn: %v, value %s\n", c.curr, *v)
			if c.filter != nil && c.filter.KeepValue(*v) {
				attrFound = true
			}

			// check the next value if we've moved on to a new span
			if c.currBufN < len(c.currBuf) {
				nextV := &c.currBuf[c.currBufN]
				if nextV.RepetitionLevel() < nextV.DefinitionLevel() {
					// we've gone to the next span, if we still haven't seen the attr then it doesn't exist
					// meaning we found the span without the attribute
					if !attrFound {
						fmt.Printf("this one does not have the attribute! %v\n", c.curr)
						return c.curr, v, nil
					}
					// reset the attrFound flag for the next span
					attrFound = false
				}
			}


			// if this is the last value then check here
			if lastBuff && c.currBufN+1 == len(c.currBuf) && !attrFound {
				fmt.Printf("this one does not have the attribute! %v\n", c.curr)
				return c.curr, v, nil
			}
			continue
		}
	}
}

func (c *NilAttributeIterator) setRowGroup(rg pq.RowGroup, min, max RowNumber, cc *ColumnChunkHelper) {
	c.closeCurrRowGroup()
	c.curr = min
	c.currRowGroup = rg
	c.currRowGroupMin = min
	c.currRowGroupMax = max
	c.currChunk = cc
}

func (c *NilAttributeIterator) setPage(pg pq.Page) {
	// Handle an outgoing page
	if c.currPage != nil {
		c.curr = c.currPageMax.Preceding() // Reposition current row number to end of this page.
		pq.Release(c.currPage)
		c.currPage = nil
	}

	// Reset value buffers
	c.currValues = nil
	c.currPageMax = EmptyRowNumber()
	c.currPageMin = EmptyRowNumber()
	c.currBufN = 0
	c.currPageN = 0

	// If we don't immediately have a new incoming page
	// then return the buffer to the pool.
	if pg == nil && c.currBuf != nil {
		nilAttributeIteratorPoolPut(c.currBuf)
		c.currBuf = nil
	}

	// Handle an incoming page
	if pg != nil {
		rn := c.curr
		rn.Skip(pg.NumRows() + 1) // Exclusive upper bound, points at the first rownumber in the next page
		c.currPage = pg
		c.currPageMin = c.curr
		c.currPageMax = rn
		c.currValues = pg.Values()
	}
}

func (c *NilAttributeIterator) closeCurrRowGroup() {
	if c.currChunk != nil {
		c.currChunk.Close()
	}

	c.currRowGroup = nil
	c.currRowGroupMin = EmptyRowNumber()
	c.currRowGroupMax = EmptyRowNumber()
	c.currChunk = nil
	c.setPage(nil)
}

func (c *NilAttributeIterator) makeResult(t RowNumber, v *pq.Value) *IteratorResult {
	// Use same static result instead of pooling
	c.at.RowNumber = t

	// The length of the Entries slice indicates if we should return the
	// value or just the row number. This has already been checked during
	// creation. NilAttributeIterator reads a single column so the slice will
	// always have length 0 or 1.
	if len(c.at.Entries) == 1 {
		if c.intern {
			c.at.Entries[0].Value = c.interner.UnsafeClone(v)
		} else {
			c.at.Entries[0].Value = v.Clone()
		}
	}

	return &c.at
}

func (c *NilAttributeIterator) Close() {
	c.closeCurrRowGroup()

	c.span.End()

	if c.intern && c.interner != nil {
		c.interner.Close()
	}
}

var nilAttributeIteratorPool = sync.Pool{
	New: func() interface{} {
		return []pq.Value{}
	},
}

func nilAttributeIteratorPoolGet(capacity, len int) []pq.Value {
	res := nilAttributeIteratorPool.Get().([]pq.Value)
	if cap(res) < capacity {
		res = make([]pq.Value, capacity)
	}
	res = res[:len]
	return res
}

func nilAttributeIteratorPoolPut(b []pq.Value) {
	for i := range b {
		b[i] = pq.Value{}
	}
	nilAttributeIteratorPool.Put(b) // nolint: staticcheck
}
