package output

import (
	"errors"
	"fmt"
)

// Format describes a printable data representation.
type Format string

const (
	FormatJSON = "json"
	FormatCSV  = "csv"
	FormatTab  = "tab"
	FormatText = "text"
)

func (f *Format) Validate() error {
	switch *f {
	case FormatJSON, FormatTab, FormatCSV, FormatText:
		return nil
	default:
		return errors.New("output format is expected to be 'json', 'tab', 'text' or 'csv'")
	}
}

func supportedFormats(data any) []Format {
	var formats []Format
	switch data.(type) {
	case Serializable, SerializableIterator:
		formats = append(formats, FormatJSON)
	case Table, TableIterator:
		formats = append(formats, FormatTab, FormatCSV)
	case Text:
		formats = append(formats, FormatText)
	}
	return formats
}

func errUnsupportedFormat(data any, f Format) error {
	supported := supportedFormats(data)

	var supportedPretty string
	for i, format := range supportedFormats(data) {
		if i > 0 {
			if i == len(supported)-1 {
				supportedPretty += " or "
			} else {
				supportedPretty += ", "
			}
		}
		supportedPretty += "'" + string(format) + "'"
	}

	return fmt.Errorf("format '%s' is not supported must be %s", f, supportedPretty)
}
