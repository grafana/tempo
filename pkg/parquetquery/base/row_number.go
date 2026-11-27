// Package base holds the types parquetquery and its pagereader share: the row number of
// a value in nested columns, the predicate and sampler an iterator applies, and the
// predicate's statistics. parquetquery aliases each of them, so callers keep using
// parquetquery.RowNumber, parquetquery.Predicate and so on.
package base

import (
	"math"
)

// RowNumber is the sequence of row numbers uniquely identifying a value
// in a tree of nested columns, starting at the top-level and including
// another row number for each level of nesting. -1 is a placeholder
// for undefined at lower levels.  RowNumbers can be compared for full
// equality using the == operator, or can be compared partially, looking
// for equal lineages down to a certain level.
// For example given the following tree, the row numbers would be:
//
//	A          0, -1, -1
//	  B        0,  0, -1
//	  C        0,  1, -1
//	    D      0,  1,  0
//	  E        0,  2, -1
//
// Currently supports 8 levels of nesting which should be enough for anybody. :)
type RowNumber [8]int32

func (t *RowNumber) Valid() bool {
	return t[0] >= 0
}

// Next increments and resets the row numbers according
// to the given repetition and definition levels. Examples
// from the Dremel whitepaper:
// https://storage.googleapis.com/pub-tools-public-publication-data/pdf/36632.pdf
// Name.Language.Country
// | value  | r | d | expected RowNumber
// |--------|---|---|-------------------
// |        |   |   | { -1, -1, -1, -1 }  <-- starting position
// | us     | 0 | 3 | {  0,  0,  0,  0 }
// | null   | 2 | 2 | {  0,  0,  1, -1 }
// | null   | 1 | 1 | {  0,  1, -1, -1 }
// | gb     | 1 | 3 | {  0,  2,  0,  0 }
// | null   | 0 | 1 | {  1,  0, -1, -1 }
func (t *RowNumber) Next(repetitionLevel, definitionLevel, maxDefinitionLevel int) {
	t[repetitionLevel]++

	// New children up through the definition level
	for i := repetitionLevel + 1; i <= definitionLevel; i++ {
		t[i] = 0
	}

	// // Children past the definition level are undefined
	for i := definitionLevel + 1; i < len(t) && i <= maxDefinitionLevel; i++ {
		t[i] = -1
	}
}

// Skip rows at the root-level.
func (t *RowNumber) Skip(numRows int64) {
	t[0] += int32(numRows)
	for i := 1; i < len(t); i++ {
		t[i] = -1
	}
}

// Preceding returns the largest representable row number that is immediately prior to this
// one. Think of it like math.NextAfter but for segmented row numbers. Examples:
//
//		RowNumber 1000.0.0 (defined at 3 levels) is preceded by 999.max.max
//	    RowNumber 1000.-1.-1 (defined at 1 level) is preceded by 999.-1.-1
func (t RowNumber) Preceding() RowNumber {
	for i := len(t) - 1; i >= 0; i-- {
		switch t[i] {
		case -1:
			continue
		case 0:
			t[i] = math.MaxInt32
		default:
			t[i]--
			return t
		}
	}
	return t
}
