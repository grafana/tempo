package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Format describes a printable data representation.
type Format string

const (
	FormatJSON = "json"
	FormatCSV  = "csv"
	FormatTab  = "tab"
)

func (f *Format) Validate() error {
	switch *f {
	case FormatJSON, FormatTab, FormatCSV:
		return nil
	default:
		return errors.New("output format is expected to be 'json', 'tab', or 'csv'")
	}
}

// A Table that can be printed / encoded in different output formats.
type Table interface {
	// Header returns the header of the table
	Header() []any
	// NextRow returns a new TableRow until the error is io.EOF
	NextRow() (TableRow, error)
}

// SerializableData represents table data that can be converted to JSON.
type SerializableData interface {
	// Data returns the table data suitable for structured data formats
	// such as json.
	Data() any
}

// A TableRow represents all data that belongs to a table row.
type TableRow interface {
	// Cells returns all table cells for this row. This is used to
	// print tabular formats such csv. The returned slice has the same
	// length as the header slice returned by the parent Table.
	Cells() []any
}

// PrintTable writes the Table data to w using the provided format.
func PrintTable(w io.Writer, f Format, data Table) error {
	switch f {
	case FormatJSON:
		return printJSON(w, data)
	case FormatTab:
		return printTab(w, data)
	case FormatCSV:
		return printCSV(w, data)
	default:
		return fmt.Errorf("format not supported yet '%s'", f)
	}
}

func printTab(w io.Writer, data Table) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	formatBuilder := strings.Builder{}
	for range data.Header() {
		formatBuilder.WriteString("%v\t")
	}
	formatBuilder.WriteRune('\n')
	format := formatBuilder.String()

	_, err := fmt.Fprintf(tw, format, data.Header()...)
	if err != nil {
		return err
	}

	row, err := data.NextRow()
	for err == nil {
		_, err = fmt.Fprintf(tw, format, row.Cells()...)
		if err != nil {
			return err
		}

		row, err = data.NextRow()
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return tw.Flush()
}

func printCSV(w io.Writer, data Table) error {
	cw := csv.NewWriter(w)
	cw.Comma = ';'

	header := data.Header()
	lineBuffer := make([]string, len(header))

	line := toStringSlice(header, lineBuffer)
	err := cw.Write(line)
	if err != nil {
		return err
	}

	row, err := data.NextRow()
	for err == nil {
		line = toStringSlice(row.Cells(), lineBuffer)
		err = cw.Write(line)
		if err != nil {
			return err
		}

		row, err = data.NextRow()
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	cw.Flush()
	return cw.Error()
}

func printJSON(w io.Writer, data Table) error {
	if serializable, ok := data.(SerializableData); ok {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(serializable.Data())
	}

	_, err := fmt.Fprintln(w, "[")
	if err != nil {
		return err
	}

	var count int
	buf := bytes.NewBuffer(make([]byte, 10240))
	row, err := data.NextRow()

	for err == nil {
		if count > 0 {
			_, err = fmt.Fprint(w, ",\n   ")
		} else {
			_, err = fmt.Fprint(w, "   ")
		}
		if err != nil {
			return err
		}
		serializableRow, ok := row.(SerializableData)
		if !ok {
			return errors.New("JSON not supported for sub command")
		}

		buf.Reset()
		err = json.NewEncoder(buf).Encode(serializableRow.Data())
		if err != nil {
			return err
		}
		buf.Truncate(buf.Len() - 1) // remove the newline

		_, err = fmt.Fprint(w, buf)
		if err != nil {
			return err
		}

		count++
		row, err = data.NextRow()
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	_, err = fmt.Println("\n]")
	return err
}

func toStringSlice(in []any, buf []string) []string {
	for i, v := range in {
		var s string
		switch v := v.(type) {
		case string:
			s = v
		case fmt.Stringer:
			s = v.String()
		default:
			s = fmt.Sprint(v)
		}

		if i < len(buf) {
			buf[i] = s
		} else {
			buf = append(buf, s)
		}
	}
	return buf[0:len(in)]
}
