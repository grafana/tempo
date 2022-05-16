//go:build go1.18

package plain

import "github.com/segmentio/parquet-go/deprecated"

// Type is a type constraint representing the possible Go types which can
// be represented by the PLAIN encoding.
type Type interface {
	bool | int32 | int64 | deprecated.Int96 | float32 | float64 | byte
}
