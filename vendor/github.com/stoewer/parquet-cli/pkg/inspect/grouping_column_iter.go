package inspect

import (
	"errors"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"
)

// newGroupingColumnIterator creates a new groupingColumnIterator.
// The iterator iterates over values of a single Parquet column and returns groups of values.
// When no groupByColumn is present, values are grouped by row. When a groupByColumn is provided,
// values are grouped such that each value group corresponds to a single value in the groupByColumn.
// The number of returned groups can be controlled via the pagination parameter.
func newGroupingColumnIterator(column, groupByColumn *parquet.Column, pagination Pagination) (*groupingColumnIterator, error) {
	it := groupingColumnIterator{
		column:        column,
		groupByColumn: groupByColumn,
		pages:         column.Pages(),
		readBuffer:    make([]parquet.Value, 1000),
		resultBuffer:  make([]parquet.Value, 1000),
		groupOffset:   pagination.Offset,
		groupLimit:    pagination.Limit,
	}
	err := it.forwardToOffset()
	if err != nil {
		return nil, fmt.Errorf("unable to create grouping column iterator: %w", err)
	}

	return &it, err
}

// groupingColumnIterator iterates over values of a single Parquet column. Values are returned in groups.
type groupingColumnIterator struct {
	column        *parquet.Column
	groupByColumn *parquet.Column
	pages         parquet.Pages
	values        parquet.ValueReader
	readBuffer    []parquet.Value
	resultBuffer  []parquet.Value
	unread        []parquet.Value
	currentGroup  int64
	groupOffset   int64
	groupLimit    *int64
}

func (r *groupingColumnIterator) Column() *parquet.Column {
	return r.column
}

func (r *groupingColumnIterator) NextGroup() ([]parquet.Value, error) {
	result := r.resultBuffer[:0]

	for {
		for i, v := range r.unread {
			if r.groupLimit != nil && r.currentGroup >= *r.groupLimit+r.groupOffset {
				return nil, fmt.Errorf("stop iteration: group limit reached: %w", io.EOF)
			}
			if r.isNewGroup(&v) && len(result) > 0 {
				r.unread = r.unread[i:]
				r.currentGroup++
				return result, nil
			}
			result = append(result, v)
		}

		count, err := r.values.ReadValues(r.readBuffer)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("unable to read values from column '%s': %w", r.column.Name(), err)
		}

		r.unread = r.readBuffer[:count]
		if len(r.unread) > 0 {
			continue
		}

		if errors.Is(err, io.EOF) {
			p, err := r.pages.ReadPage()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return nil, fmt.Errorf("unable to read new page from column '%s': %w", r.column.Name(), err)
				}
				if len(result) > 0 {
					return result, nil
				}
				return nil, err
			}
			r.values = p.Values()
		}
	}
}

func (r *groupingColumnIterator) forwardToOffset() error {
	for {
		page, err := r.pages.ReadPage()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return fmt.Errorf("unable to read new page from column '%s': %w", r.column.Name(), err)
			}
			return err
		}

		if r.currentGroup+page.NumRows() >= r.groupOffset {
			r.values = page.Values()
			break
		}

		r.currentGroup += page.NumRows()
	}

	for {
		for i, v := range r.unread {
			if r.isNewGroup(&v) && i > 0 {
				r.currentGroup++
			}
			if r.currentGroup >= r.groupOffset {
				r.unread = r.unread[i:]
				return nil
			}
		}

		count, err := r.values.ReadValues(r.readBuffer)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("unable to read values from column '%s': %w", r.column.Name(), err)
		}

		r.unread = r.readBuffer[:count]

		if errors.Is(err, io.EOF) && count <= 0 {
			return err
		}
	}
}

func (r *groupingColumnIterator) isNewGroup(v *parquet.Value) bool {
	if r.groupByColumn == nil {
		// no group by column: each now row is a new group
		return v.RepetitionLevel() == 0
	}
	if r.column.Index() == r.groupByColumn.Index() {
		// column equals group by column: each new value is a new group
		return true
	}

	// a value belongs to new group when it repeats at the same level or
	// lower as the definition value of the group by column
	return v.RepetitionLevel() <= r.groupByColumn.MaxDefinitionLevel()
}
