package inspect

import (
	"fmt"

	"github.com/parquet-go/parquet-go"
	"github.com/stoewer/parquet-cli/pkg/output"
)

var (
	rowCellFields = [...]string{"size", "values", "nulls"}
)

type RowCellStats struct {
	Column string `json:"col"`
	Size   int    `json:"size"`
	Values int    `json:"values"`
	Nulls  int    `json:"nulls"`
}

type RowStats struct {
	RowNumber int
	Stats     []RowCellStats
}

func (rs *RowStats) SerializableData() any {
	return rs.Stats
}

func (rs *RowStats) Cells() []any {
	cells := make([]any, 0, len(rs.Stats)*len(rowCellFields)+1)
	cells = append(cells, rs.RowNumber)
	for _, c := range rs.Stats {
		cells = append(cells, c.Size, c.Values, c.Nulls)
	}
	return cells
}

type RowStatOptions struct {
	Pagination
	Columns []int
}

func NewRowStatCalculator(file *parquet.File, options RowStatOptions) (*RowStatCalculator, error) {
	all := LeafColumns(file)
	var columns []*parquet.Column

	if len(options.Columns) == 0 {
		columns = all
	} else {
		columns = make([]*parquet.Column, 0, len(options.Columns))
		for _, idx := range options.Columns {
			if idx >= len(all) {
				return nil, fmt.Errorf("column index expectd be below %d but was %d", idx, len(all))
			}
			columns = append(columns, all[idx])
		}
	}

	c := RowStatCalculator{
		header:     make([]any, 0, len(columns)*len(rowCellFields)+1),
		columnIter: make([]*groupingColumnIterator, 0, len(columns)),
	}

	c.header = append(c.header, "Row")
	for _, col := range columns {
		it, err := newGroupingColumnIterator(col, nil, options.Pagination)
		if err != nil {
			return nil, fmt.Errorf("unable to create row stats calculator: %w", err)
		}
		c.columnIter = append(c.columnIter, it)
		c.header = append(c.header, fmt.Sprintf("%d/%s: %s", col.Index(), col.Name(), rowCellFields[0]), rowCellFields[1], rowCellFields[2])
	}

	return &c, nil
}

type RowStatCalculator struct {
	header     []any
	columnIter []*groupingColumnIterator
	rowNumber  int
}

func (c *RowStatCalculator) Header() []any {
	return c.header
}

func (c *RowStatCalculator) NextRow() (output.TableRow, error) {
	rs := RowStats{
		RowNumber: c.rowNumber,
		Stats:     make([]RowCellStats, 0, len(c.columnIter)),
	}

	for _, it := range c.columnIter {
		values, err := it.NextGroup()
		if err != nil {
			return nil, err
		}
		cs := RowCellStats{Column: it.Column().Name()}
		for _, val := range values {
			if val.IsNull() {
				cs.Nulls++
			} else {
				cs.Values++
				cs.Size += len(val.Bytes())
			}
		}
		rs.Stats = append(rs.Stats, cs)
	}

	c.rowNumber++
	return &rs, nil
}
