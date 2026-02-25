package inspect

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/stoewer/parquet-cli/pkg/output"

	"github.com/parquet-go/parquet-go"
)

type PageInfoOptions struct {
	Pagination
	Column int
}

type PageInfo struct {
	Pagination
	column       int
	rowGroups    []parquet.RowGroup
	offsetIndex  parquet.OffsetIndex
	pages        parquet.Pages
	currRowGroup int
	currPage     int
	pageOffset   int
}

func NewPageInfo(file *parquet.File, opt PageInfoOptions) (*PageInfo, error) {
	all := LeafColumns(file)
	if opt.Column < 0 || opt.Column >= len(all) {
		return nil, fmt.Errorf("column index expectd between 0 and %d, but was %d", len(all)-1, opt.Column)
	}

	// select row groups according to offset
	rowGroups := file.RowGroups()
	var (
		currRowGroup int
		currPage     int
		pageOffset   int
	)
	for currRowGroup < len(rowGroups) {
		ci, err := rowGroups[currRowGroup].ColumnChunks()[opt.Column].ColumnIndex()
		if err != nil {
			return nil, err
		}

		if currPage+ci.NumPages() > int(opt.Offset) {
			break
		}

		currPage += ci.NumPages()
		pageOffset = currPage
		currRowGroup++
	}

	if currRowGroup >= len(rowGroups) {
		return nil, errors.New("no row groups / pages left")
	}

	// forward to the correct page
	pages := rowGroups[currRowGroup].ColumnChunks()[opt.Column].Pages()
	for currPage < int(opt.Offset) {
		currPage++
		_, err := pages.ReadPage()
		if err != nil {
			return nil, err
		}
	}

	offsetIndex, err := rowGroups[currRowGroup].ColumnChunks()[opt.Column].OffsetIndex()
	if err != nil {
		return nil, err
	}

	return &PageInfo{
		Pagination:   opt.Pagination,
		column:       opt.Column,
		rowGroups:    rowGroups,
		offsetIndex:  offsetIndex,
		pages:        pages,
		currRowGroup: currRowGroup,
		currPage:     currPage,
		pageOffset:   pageOffset,
	}, nil
}

func (p *PageInfo) Header() []any {
	return []any{"Row group", "Page", "Size", "Compressed size", "Rows", "Values", "Nulls", "Min val", "Max val"}
}

func (p *PageInfo) NextRow() (output.TableRow, error) {
	if p.currRowGroup >= len(p.rowGroups) || (p.Limit != nil && p.currPage >= int(p.Offset+*p.Limit)) {
		return nil, io.EOF
	}

	page, err := p.pages.ReadPage()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, err
		}

		p.currRowGroup++
		if p.currRowGroup >= len(p.rowGroups) {
			return nil, io.EOF
		}

		p.pageOffset += p.offsetIndex.NumPages()
		p.pages = p.rowGroups[p.currRowGroup].ColumnChunks()[p.column].Pages()
		p.offsetIndex, err = p.rowGroups[p.currRowGroup].ColumnChunks()[p.column].OffsetIndex()
		if err != nil {
			return nil, err
		}

		return p.NextRow()
	}

	minVal, maxVal, ok := page.Bounds()

	pl := PageLine{
		RowGroup:       p.currRowGroup,
		Page:           p.currPage,
		Size:           page.Size(),
		CompressedSize: p.offsetIndex.CompressedPageSize(p.currPage - p.pageOffset),
		NumRows:        page.NumRows(),
		NumValues:      page.NumValues(),
		NumNulls:       page.NumNulls(),
	}
	if ok {
		pl.MinVal = sanitizeString(minVal.String(), 40)
		pl.MaxVal = sanitizeString(maxVal.String(), 40)
	}

	p.currPage++
	return &pl, nil
}

func (p *PageInfo) NextSerializable() (any, error) {
	return p.NextRow()
}

type PageLine struct {
	RowGroup       int    `json:"row_group"`
	Page           int    `json:"page"`
	Size           int64  `json:"size"`
	CompressedSize int64  `json:"compressed_size"`
	NumRows        int64  `json:"num_rows"`
	NumValues      int64  `json:"num_values"`
	NumNulls       int64  `json:"num_nulls"`
	MinVal         string `json:"min_val"`
	MaxVal         string `json:"max_val"`
}

func (p *PageLine) Cells() []any {
	return []any{p.RowGroup, p.Page, p.Size, p.CompressedSize, p.NumRows, p.NumValues, p.NumNulls, p.MinVal, p.MaxVal}
}

func sanitizeString(s string, limit int) string {
	for i, r := range s {
		if !unicode.IsPrint(r) {
			return "<not printable>"
		}
		if i >= limit {
			break
		}
	}
	if len(s) <= limit {
		return s
	}
	s = s[:limit] + "..."

	return strings.ReplaceAll(s, "\n", " ")
}
