package parquet

import "errors"

var (
	// ErrCorrupted is an error returned by the Err method of ColumnPages
	// instances when they encountered a mismatch between the CRC checksum
	// recorded in a page header and the one computed while reading the page
	// data.
	ErrCorrupted = errors.New("corrupted parquet page")

	// ErrMissingRootColumn is an error returned when opening an invalid parquet
	// file which does not have a root column.
	ErrMissingRootColumn = errors.New("parquet file is missing a root column")

	// ErrRowGroupSchemaMissing is an error returned when attempting to write a
	// row group but the source has no schema.
	ErrRowGroupSchemaMissing = errors.New("cannot write rows to a row group which has no schema")

	// ErrRowGroupSchemaMismatch is an error returned when attempting to write a
	// row group but the source and destination schemas differ.
	ErrRowGroupSchemaMismatch = errors.New("cannot write row groups with mismatching schemas")

	// ErrRowGroupSortingColumnsMismatch is an error returned when attempting to
	// write a row group but the sorting columns differ in the source and
	// destination.
	ErrRowGroupSortingColumnsMismatch = errors.New("cannot write row groups with mismatching sorting columns")

	// ErrSeekOutOfRange is an error returned when seeking to a row index which
	// is less than the first row of a page.
	ErrSeekOutOfRange = errors.New("seek to row index out of page range")
)
