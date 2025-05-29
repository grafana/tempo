package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// PrintTable writes the TableIterator data to w using the provided format.
func PrintTable(w io.Writer, f Format, data TableIterator) error {
	switch f {
	case FormatJSON:
		return printTableToJSON(w, data)
	case FormatTab:
		return printTab(w, data)
	case FormatCSV:
		return printCSV(w, data)
	default:
		return fmt.Errorf("format not supported yet '%s'", f)
	}
}

func printTableToJSON(w io.Writer, data TableIterator) error {
	if serializable, ok := data.(Serializable); ok {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(serializable.SerializableData())
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
		serializableRow, ok := row.(Serializable)
		if !ok {
			return errors.New("JSON not supported for sub command")
		}

		buf.Reset()
		err = json.NewEncoder(buf).Encode(serializableRow.SerializableData())
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
