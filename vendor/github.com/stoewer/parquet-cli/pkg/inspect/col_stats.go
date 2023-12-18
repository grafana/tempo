package inspect

import (
	"errors"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"
	"github.com/stoewer/parquet-cli/pkg/output"
)

var (
	columnStatHeader = [...]any{
		"Index",
		"Name",
		"Max Def",
		"Max Rep",
		"Size",
		"Compressed size",
		"Pages",
		"Rows",
		"Page min rows",
		"Page max rows",
		"Values",
		"Page min vals",
		"Page max vals",
		"Nulls",
		"Page min nulls",
		"Page max nulls",
	}
)

type ColumnStats struct {
	Index          int    `json:"index"`
	Name           string `json:"name"`
	MaxDef         int    `json:"maxDef"`
	MaxRep         int    `json:"maxRep"`
	Size           int64  `json:"size"`
	CompressedSize int64  `json:"compressedSize"`
	Pages          int    `json:"pages"`
	Rows           int64  `json:"rows"`
	PageMinRows    int64  `json:"pageMinRows"`
	PageMaxRows    int64  `json:"pageMaxRows"`
	Values         int64  `json:"values"`
	PageMinValues  int64  `json:"pageMinValues"`
	PageMaxValues  int64  `json:"pageMaxValues"`
	Nulls          int64  `json:"nulls"`
	PageMinNulls   int64  `json:"pageMinNulls"`
	PageMaxNulls   int64  `json:"pageMaxNulls"`

	cells []any
}

func (rs *ColumnStats) Data() any {
	return rs
}

func (rs *ColumnStats) Cells() []any {
	if rs.cells == nil {
		rs.cells = []any{
			rs.Index,
			rs.Name,
			rs.MaxDef,
			rs.MaxRep,
			rs.Size,
			rs.CompressedSize,
			rs.Pages,
			rs.Rows,
			rs.PageMinRows,
			rs.PageMaxRows,
			rs.Values,
			rs.PageMinValues,
			rs.PageMaxValues,
			rs.Nulls,
			rs.PageMinNulls,
			rs.PageMaxNulls,
		}
	}
	return rs.cells
}

func NewColStatCalculator(file *parquet.File, selectedCols []int) (*ColStatCalculator, error) {
	all := LeafColumns(file)
	var columns []*parquet.Column

	if len(selectedCols) == 0 {
		columns = all
	} else {
		columns = make([]*parquet.Column, 0, len(selectedCols))
		for _, idx := range selectedCols {
			if idx >= len(all) {
				return nil, fmt.Errorf("column index expectd be below %d but was %d", idx, len(all))
			}
			columns = append(columns, all[idx])
		}
	}

	return &ColStatCalculator{file: file, columns: columns}, nil
}

type ColStatCalculator struct {
	file    *parquet.File
	columns []*parquet.Column
	current int
}

func (cc *ColStatCalculator) Header() []any {
	return columnStatHeader[:]
}

func (cc *ColStatCalculator) NextRow() (output.TableRow, error) {
	if cc.current >= len(cc.columns) {
		return nil, fmt.Errorf("stop iteration: no more culumns: %w", io.EOF)
	}

	col := cc.columns[cc.current]
	cc.current++
	stats := ColumnStats{
		Index:  col.Index(),
		Name:   col.Name(),
		MaxDef: col.MaxDefinitionLevel(),
		MaxRep: col.MaxRepetitionLevel(),
	}

	for _, rg := range cc.file.RowGroups() {
		chunk := rg.ColumnChunks()[col.Index()]

		index, err := chunk.OffsetIndex()
		if err != nil {
			return nil, fmt.Errorf("unable to read offset index from column '%s': %w", col.Name(), err)
		}
		if index != nil {
			for i := 0; i < index.NumPages(); i++ {
				stats.CompressedSize += index.CompressedPageSize(i)
			}
		}

		pages := chunk.Pages()
		page, err := pages.ReadPage()
		for err == nil {
			stats.Pages++
			stats.Size += page.Size()
			stats.Rows += page.NumRows()
			stats.PageMinRows = min(stats.PageMinRows, page.NumRows())
			stats.PageMaxRows = max(stats.PageMaxRows, page.NumRows())
			stats.Values += page.NumValues()
			stats.PageMinValues = min(stats.PageMinValues, page.NumRows())
			stats.PageMaxValues = max(stats.PageMaxValues, page.NumRows())
			stats.Nulls += page.NumNulls()
			stats.PageMinNulls = min(stats.PageMinNulls, page.NumNulls())
			stats.PageMaxNulls = max(stats.PageMaxNulls, page.NumNulls())
			page, err = pages.ReadPage()
		}
		if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("unable to read page rom column '%s': %w", col.Name(), err)
		}
	}

	return &stats, nil
}
