// Package tw defines types and constants for table formatting and configuration,
// including validation logic for various table properties.
package tw

import "github.com/olekukonko/errors" // Custom error handling library

// Position defines where formatting applies in the table (e.g., header, footer, or rows).
type Position string

// Validate checks if the Position is one of the allowed values: Header, Footer, or Row.
func (pos Position) Validate() error {
	switch pos {
	case Header, Footer, Row:
		return nil // Valid position
	}
	// Return an error for any unrecognized position
	return errors.New("invalid position")
}

// Filter defines a function type for processing cell content.
// It takes a slice of strings (representing cell data) and returns a processed slice.
type Filter func([]string) []string

// Formatter defines an interface for types that can format themselves into a string.
// Used for custom formatting of table cell content.
type Formatter interface {
	Format() string // Returns the formatted string representation
}

// Align specifies the text alignment within a table cell.
type Align string

// Validate checks if the Align is one of the allowed values: None, Center, Left, or Right.
func (a Align) Validate() error {
	switch a {
	case AlignNone, AlignCenter, AlignLeft, AlignRight:
		return nil // Valid alignment
	}
	// Return an error for any unrecognized alignment
	return errors.New("invalid align")
}

// Level indicates the vertical position of a line in the table (e.g., header, body, or footer).
type Level int

// Validate checks if the Level is one of the allowed values: Header, Body, or Footer.
func (l Level) Validate() error {
	switch l {
	case LevelHeader, LevelBody, LevelFooter:
		return nil // Valid level
	}
	// Return an error for any unrecognized level
	return errors.New("invalid level")
}

// Location specifies the horizontal position of a cell or column within a table row.
type Location string

// Validate checks if the Location is one of the allowed values: First, Middle, or End.
func (l Location) Validate() error {
	switch l {
	case LocationFirst, LocationMiddle, LocationEnd:
		return nil // Valid location
	}
	// Return an error for any unrecognized location
	return errors.New("invalid location")
}
