package inspect

import (
	"errors"
	"fmt"
	"io"
	"strings"

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
		"Size Ratio",
		"Pages",
		"Rows",
		"Values",
		"Nulls",
		"Path",
	}
	columnStatHeaderFull = [...]any{
		"Index",
		"Name",
		"Max Def",
		"Max Rep",
		"Size",
		"Compressed size",
		"Size Ratio",
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
		"Path",
	}
)

type ColumnStats struct {
	Index          int     `json:"index"`
	Name           string  `json:"name"`
	MaxDef         int     `json:"max_def"`
	MaxRep         int     `json:"max_rep"`
	Size           int64   `json:"size"`
	CompressedSize int64   `json:"compressed_size"`
	SizeRatio      float64 `json:"size_ratio"`
	Pages          int     `json:"pages"`
	Rows           int64   `json:"rows"`
	Values         int64   `json:"values"`
	Nulls          int64   `json:"nulls"`
	Path           string  `json:"path"`
}

func (rs *ColumnStats) Cells() []any {
	return []any{
		rs.Index,
		rs.Name,
		rs.MaxDef,
		rs.MaxRep,
		rs.Size,
		rs.CompressedSize,
		fmt.Sprintf("%.3f", rs.SizeRatio),
		rs.Pages,
		rs.Rows,
		rs.Values,
		rs.Nulls,
		rs.Path,
	}
}

type ColumnStatsFull struct {
	ColumnStats
	PageMinRows   int64 `json:"page_min_rows"`
	PageMaxRows   int64 `json:"page_max_rows"`
	PageMinValues int64 `json:"page_min_values"`
	PageMaxValues int64 `json:"page_max_values"`
	PageMinNulls  int64 `json:"page_min_nulls"`
	PageMaxNulls  int64 `json:"page_max_nulls"`
}

func (rs *ColumnStatsFull) Cells() []any {
	return []any{
		rs.Index,
		rs.Name,
		rs.MaxDef,
		rs.MaxRep,
		rs.Size,
		rs.CompressedSize,
		fmt.Sprintf("%.3f", rs.SizeRatio),
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
		rs.Path,
	}
}

func NewColStatCalculator(file *parquet.File, selectedCols []int, verbose bool) (*ColStatCalculator, error) {
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

	return &ColStatCalculator{file: file, columns: columns, verbose: verbose}, nil
}

type ColStatCalculator struct {
	file    *parquet.File
	verbose bool
	columns []*parquet.Column
	current int
}

func (cc *ColStatCalculator) Header() []any {
	if cc.verbose {
		return columnStatHeaderFull[:]
	}
	return columnStatHeader[:]
}

func (cc *ColStatCalculator) NextRow() (output.TableRow, error) {
	if cc.current >= len(cc.columns) {
		return nil, fmt.Errorf("stop iteration: no more culumns: %w", io.EOF)
	}

	col := cc.columns[cc.current]
	cc.current++
	stats := ColumnStatsFull{
		ColumnStats: ColumnStats{
			Index:  col.Index(),
			Name:   PathToDisplayName(col.Path()),
			MaxDef: col.MaxDefinitionLevel(),
			MaxRep: col.MaxRepetitionLevel(),
		},
	}

	totalSize := cc.file.Size()

	for _, rg := range cc.file.RowGroups() {
		chunk := rg.ColumnChunks()[col.Index()]

		index, err := chunk.OffsetIndex()
		if err == nil && index != nil {
			for i := 0; i < index.NumPages(); i++ {
				stats.CompressedSize += index.CompressedPageSize(i)
			}
		} else {
			stats.CompressedSize = 0 // Prevents a crash if the OffsetIndex is not present
		}

		path := strings.Join(col.Path(), ".")

		pages := chunk.Pages()
		page, err := pages.ReadPage()
		for err == nil {
			stats.Pages++
			stats.Size += page.Size()
			stats.Rows += page.NumRows()
			stats.Values += page.NumValues()
			stats.Nulls += page.NumNulls()

			stats.PageMinNulls = min(stats.PageMinNulls, page.NumNulls())
			stats.PageMaxNulls = max(stats.PageMaxNulls, page.NumNulls())
			stats.PageMinValues = min(stats.PageMinValues, page.NumRows())
			stats.PageMaxValues = max(stats.PageMaxValues, page.NumRows())
			stats.PageMinRows = min(stats.PageMinRows, page.NumRows())
			stats.PageMaxRows = max(stats.PageMaxRows, page.NumRows())

			stats.Path = path

			page, err = pages.ReadPage()
		}

		if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("unable to read page rom column '%s': %w", col.Name(), err)
		}
	}

	if totalSize > 0 {
		stats.SizeRatio = float64(stats.CompressedSize) / float64(totalSize)
	}

	if cc.verbose {
		return &stats, nil
	}
	return &stats.ColumnStats, nil
}

func (cc *ColStatCalculator) NextSerializable() (any, error) {
	return cc.NextRow()
}
