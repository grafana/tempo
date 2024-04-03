// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package parseutils // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/parseutils"

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ReadCSVRow reads a CSV row from the csv reader, returning the fields parsed from the line.
// We make the assumption that the payload we are reading is a single row, so we allow newline characters in fields.
// However, the csv package does not support newlines in a CSV field (it assumes rows are newline separated),
// so in order to support parsing newlines in a field, we need to stitch together the results of multiple Read calls.
func ReadCSVRow(row string, delimiter rune, lazyQuotes bool) ([]string, error) {
	reader := csv.NewReader(strings.NewReader(row))
	reader.Comma = delimiter
	// -1 indicates a variable length of fields
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = lazyQuotes

	lines := make([][]string, 0, 1)
	for {
		line, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil && len(line) == 0 {
			return nil, fmt.Errorf("read csv line: %w", err)
		}

		lines = append(lines, line)
	}

	// If the input is empty, we might not get any lines
	if len(lines) == 0 {
		return nil, errors.New("no csv lines found")
	}

	/*
		This parser is parsing a single value, which came from a single log entry.
		Therefore, if there are multiple lines here, it should be assumed that each
		subsequent line contains a continuation of the last field in the previous line.

		Given a file w/ headers "A,B,C,D,E" and contents "aa,b\nb,cc,d\nd,ee",
		expect reader.Read() to return bodies:
		- ["aa","b"]
		- ["b","cc","d"]
		- ["d","ee"]
	*/

	joinedLine := lines[0]
	for i := 1; i < len(lines); i++ {
		nextLine := lines[i]

		// The first element of the next line is a continuation of the previous line's last element
		joinedLine[len(joinedLine)-1] += "\n" + nextLine[0]

		// The remainder are separate elements
		for n := 1; n < len(nextLine); n++ {
			joinedLine = append(joinedLine, nextLine[n])
		}
	}

	return joinedLine, nil
}

// MapCSVHeaders creates a map of headers[i] -> fields[i].
func MapCSVHeaders(headers []string, fields []string) (map[string]any, error) {
	if len(fields) != len(headers) {
		return nil, fmt.Errorf("wrong number of fields: expected %d, found %d", len(headers), len(fields))
	}

	parsedValues := make(map[string]any, len(headers))

	for i, val := range fields {
		parsedValues[headers[i]] = val
	}

	return parsedValues, nil
}
