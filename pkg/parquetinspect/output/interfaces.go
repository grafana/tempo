package output

// A Table represents a tabular data that can also be printed as CSV.
// Suitable for small tables that fit into memory.
type Table interface {
	Header() []string
	Rows() []TableRow
}

// A TableIterator that can efficiently be printed as large table or CSV.
// Suitable for larger tables that do not fit into memory.
type TableIterator interface {
	// Header returns the header of the table
	Header() []any
	// NextRow returns a new TableRow until the error is io.EOF
	NextRow() (TableRow, error)
}

// A TableRow represents all data that belongs to a table row.
type TableRow interface {
	// Cells returns all table cells for this row. This is used to
	// print tabular formats such csv. The returned slice has the same
	// length as the header slice returned by the parent TableIterator.
	Cells() []any
}

// Serializable represents data that can be converted to JSON or YAML.
type Serializable interface {
	// SerializableData returns arbitrary data that can be converted to formats like JSON or YAML.
	SerializableData() any
}

// SerializableIterator represents a stream of data that can be converted to JSON or YAML.
type SerializableIterator interface {
	NextSerializable() (any, error)
}

// Text represents a multi line text that can be printed but is not a table or another
// structured format such as JSON or YAML.
type Text interface {
	Text() (string, error)
}
