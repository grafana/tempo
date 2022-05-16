package parquet

import (
	"fmt"
	"math"
)

const (
	// MaxColumnDepth is the maximum column depth supported by this package.
	MaxColumnDepth = math.MaxInt8

	// MaxColumnIndex is the maximum column index supported by this package.
	MaxColumnIndex = math.MaxInt16

	// MaxRepetitionLevel is the maximum repetition level supported by this package.
	MaxRepetitionLevel = math.MaxInt8

	// MaxDefinitionLevel is the maximum definition level supported by this package.
	MaxDefinitionLevel = math.MaxInt8
)

func makeRepetitionLevel(i int) int8 {
	checkIndexRange("repetition level", i, 0, MaxRepetitionLevel)
	return int8(i)
}

func makeDefinitionLevel(i int) int8 {
	checkIndexRange("definition level", i, 0, MaxDefinitionLevel)
	return int8(i)
}

func makeColumnIndex(i int) int16 {
	checkIndexRange("column index", i, 0, MaxColumnIndex)
	return int16(i)
}

func checkIndexRange(typ string, i, min, max int) {
	if i < min || i > max {
		panic(errIndexOutOfRange(typ, i, min, max))
	}
}

func errIndexOutOfRange(typ string, i, min, max int) error {
	return fmt.Errorf("%s out of range: %d not in [%d:%d]", typ, i, min, max)
}
