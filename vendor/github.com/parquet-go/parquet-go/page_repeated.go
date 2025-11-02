package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type repeatedPage struct {
	base               Page
	maxRepetitionLevel byte
	maxDefinitionLevel byte
	definitionLevels   []byte
	repetitionLevels   []byte
}

func newRepeatedPage(base Page, maxRepetitionLevel, maxDefinitionLevel byte, repetitionLevels, definitionLevels []byte) *repeatedPage {
	return &repeatedPage{
		base:               base,
		maxRepetitionLevel: maxRepetitionLevel,
		maxDefinitionLevel: maxDefinitionLevel,
		definitionLevels:   definitionLevels,
		repetitionLevels:   repetitionLevels,
	}
}

func (page *repeatedPage) Type() Type { return page.base.Type() }

func (page *repeatedPage) Column() int { return page.base.Column() }

func (page *repeatedPage) Dictionary() Dictionary { return page.base.Dictionary() }

func (page *repeatedPage) NumRows() int64 { return int64(countLevelsEqual(page.repetitionLevels, 0)) }

func (page *repeatedPage) NumValues() int64 { return int64(len(page.definitionLevels)) }

func (page *repeatedPage) NumNulls() int64 {
	return int64(countLevelsNotEqual(page.definitionLevels, page.maxDefinitionLevel))
}

func (page *repeatedPage) Bounds() (min, max Value, ok bool) { return page.base.Bounds() }

func (page *repeatedPage) Size() int64 {
	return int64(len(page.repetitionLevels)) + int64(len(page.definitionLevels)) + page.base.Size()
}

func (page *repeatedPage) RepetitionLevels() []byte { return page.repetitionLevels }

func (page *repeatedPage) DefinitionLevels() []byte { return page.definitionLevels }

func (page *repeatedPage) Data() encoding.Values { return page.base.Data() }

func (page *repeatedPage) Values() ValueReader {
	return &repeatedPageValues{
		page:   page,
		values: page.base.Values(),
	}
}

func (page *repeatedPage) Slice(i, j int64) Page {
	numRows := page.NumRows()
	if i < 0 || i > numRows {
		panic(errPageBoundsOutOfRange(i, j, numRows))
	}
	if j < 0 || j > numRows {
		panic(errPageBoundsOutOfRange(i, j, numRows))
	}
	if i > j {
		panic(errPageBoundsOutOfRange(i, j, numRows))
	}

	maxRepetitionLevel := page.maxRepetitionLevel
	maxDefinitionLevel := page.maxDefinitionLevel
	repetitionLevels := page.repetitionLevels
	definitionLevels := page.definitionLevels

	rowIndex0 := 0
	rowIndex1 := len(repetitionLevels)
	rowIndex2 := len(repetitionLevels)

	for k, def := range repetitionLevels {
		if def == 0 {
			if rowIndex0 == int(i) {
				rowIndex1 = k
				break
			}
			rowIndex0++
		}
	}

	for k, def := range repetitionLevels[rowIndex1:] {
		if def == 0 {
			if rowIndex0 == int(j) {
				rowIndex2 = rowIndex1 + k
				break
			}
			rowIndex0++
		}
	}

	numNulls1 := countLevelsNotEqual(definitionLevels[:rowIndex1], maxDefinitionLevel)
	numNulls2 := countLevelsNotEqual(definitionLevels[rowIndex1:rowIndex2], maxDefinitionLevel)

	i = int64(rowIndex1 - numNulls1)
	j = int64(rowIndex2 - (numNulls1 + numNulls2))

	return newRepeatedPage(
		page.base.Slice(i, j),
		maxRepetitionLevel,
		maxDefinitionLevel,
		repetitionLevels[rowIndex1:rowIndex2:rowIndex2],
		definitionLevels[rowIndex1:rowIndex2:rowIndex2],
	)
}
