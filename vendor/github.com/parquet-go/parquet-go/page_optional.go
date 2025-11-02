package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type optionalPage struct {
	base               Page
	maxDefinitionLevel byte
	definitionLevels   []byte
}

func newOptionalPage(base Page, maxDefinitionLevel byte, definitionLevels []byte) *optionalPage {
	return &optionalPage{
		base:               base,
		maxDefinitionLevel: maxDefinitionLevel,
		definitionLevels:   definitionLevels,
	}
}

func (page *optionalPage) Type() Type { return page.base.Type() }

func (page *optionalPage) Column() int { return page.base.Column() }

func (page *optionalPage) Dictionary() Dictionary { return page.base.Dictionary() }

func (page *optionalPage) NumRows() int64 { return int64(len(page.definitionLevels)) }

func (page *optionalPage) NumValues() int64 { return int64(len(page.definitionLevels)) }

func (page *optionalPage) NumNulls() int64 {
	return int64(countLevelsNotEqual(page.definitionLevels, page.maxDefinitionLevel))
}

func (page *optionalPage) Bounds() (min, max Value, ok bool) { return page.base.Bounds() }

func (page *optionalPage) Size() int64 { return int64(len(page.definitionLevels)) + page.base.Size() }

func (page *optionalPage) RepetitionLevels() []byte { return nil }

func (page *optionalPage) DefinitionLevels() []byte { return page.definitionLevels }

func (page *optionalPage) Data() encoding.Values { return page.base.Data() }

func (page *optionalPage) Values() ValueReader {
	return &optionalPageValues{
		page:   page,
		values: page.base.Values(),
	}
}

func (page *optionalPage) Slice(i, j int64) Page {
	maxDefinitionLevel := page.maxDefinitionLevel
	definitionLevels := page.definitionLevels
	numNulls1 := int64(countLevelsNotEqual(definitionLevels[:i], maxDefinitionLevel))
	numNulls2 := int64(countLevelsNotEqual(definitionLevels[i:j], maxDefinitionLevel))
	return newOptionalPage(
		page.base.Slice(i-numNulls1, j-(numNulls1+numNulls2)),
		maxDefinitionLevel,
		definitionLevels[i:j:j],
	)
}
