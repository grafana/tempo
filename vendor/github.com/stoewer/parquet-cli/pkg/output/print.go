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
	"unsafe"
)

type PrintOptions struct {
	Format Format
	Color  bool
}

func Print(out io.Writer, data any, opts *PrintOptions) error {
	switch opts.Format {
	case FormatText:
		if text, ok := data.(Text); ok {
			return printText(out, text)
		}
	case FormatTab:
		if table, ok := data.(TableIterator); ok {
			return printTab(out, table)
		}
	case FormatCSV:
		if table, ok := data.(TableIterator); ok {
			return printCSV(out, table)
		}
	case FormatJSON:
		if ser, ok := data.(SerializableIterator); ok {
			return printJSON(out, ser)
		}
	}
	return errUnsupportedFormat(data, opts.Format)
}

func printJSON(w io.Writer, data SerializableIterator) error {
	_, err := fmt.Fprintln(w, "[")
	if err != nil {
		return err
	}

	var count int
	buf := bytes.NewBuffer(make([]byte, 10240))
	next, err := data.NextSerializable()

	for err == nil {
		if count > 0 {
			_, err = fmt.Fprint(w, ",\n   ")
		} else {
			_, err = fmt.Fprint(w, "   ")
		}
		if err != nil {
			return err
		}

		buf.Reset()
		err = json.NewEncoder(buf).Encode(next)
		if err != nil {
			return err
		}
		buf.Truncate(buf.Len() - 1) // remove the newline

		_, err = fmt.Fprint(w, buf)
		if err != nil {
			return err
		}

		count++
		next, err = data.NextSerializable()
	}
	if !errors.Is(err, io.EOF) {
		return err
	}

	_, err = fmt.Println("\n]")
	return err
}

func printTab(w io.Writer, data TableIterator) error {
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
	if !errors.Is(err, io.EOF) {
		return err
	}

	return tw.Flush()
}

func printCSV(w io.Writer, data TableIterator) error {
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
	if !errors.Is(err, io.EOF) {
		return err
	}

	cw.Flush()
	return cw.Error()
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

func printText(out io.Writer, data Text) error {
	s, err := data.Text()
	if err != nil {
		return fmt.Errorf("unable to print text: %w", err)
	}

	b := unsafe.Slice(unsafe.StringData(s), len(s))

	_, err = out.Write(b)
	return err
}
