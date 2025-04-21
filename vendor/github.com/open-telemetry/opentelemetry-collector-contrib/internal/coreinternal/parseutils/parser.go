// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package parseutils // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/parseutils"

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/multierr"
)

// SplitString will split the input on the delimiter and return the resulting slice while respecting quotes. Outer quotes are stripped.
// Use in place of `strings.Split` when quotes need to be respected.
// Requires `delimiter` not be an empty string
func SplitString(input, delimiter string) ([]string, error) {
	var result []string
	current := ""
	delimiterLength := len(delimiter)
	quoteChar := "" // "" means we are not in quotes
	escaped := false

	for i := 0; i < len(input); i++ {
		if quoteChar == "" && i+delimiterLength <= len(input) && input[i:i+delimiterLength] == delimiter { // delimiter
			if current == "" { // leading || trailing delimiter; ignore
				i += delimiterLength - 1
				continue
			}
			result = append(result, current)
			current = ""
			i += delimiterLength - 1
			continue
		}

		if !escaped { // consider quote termination so long as previous character wasn't backslash
			if quoteChar == "" && (input[i] == '"' || input[i] == '\'') { // start of quote
				quoteChar = string(input[i])
				continue
			}
			if string(input[i]) == quoteChar { // end of quote
				quoteChar = ""
				continue
			}
			// Only if we weren't escaped could the next character result in escaped state
			escaped = input[i] == '\\' // potentially escaping next character
		} else {
			escaped = false
		}

		current += string(input[i])
	}

	if quoteChar != "" { // check for closed quotes
		return nil, errors.New("never reached the end of a quoted value")
	}
	if current != "" { // avoid adding empty value bc of a trailing delimiter
		return append(result, current), nil
	}

	return result, nil
}

// ParseKeyValuePairs will split each string in `pairs` on the `delimiter` into a key and value string that get added to a map and returned.
func ParseKeyValuePairs(pairs []string, delimiter string) (map[string]any, error) {
	parsed := make(map[string]any)
	var err error
	for _, p := range pairs {
		pair := strings.SplitN(p, delimiter, 2)
		if len(pair) != 2 {
			err = multierr.Append(err, fmt.Errorf("cannot split %q into 2 items, got %d item(s)", p, len(pair)))
			continue
		}

		key := strings.TrimSpace(pair[0])
		value := strings.TrimSpace(pair[1])

		parsed[key] = value
	}
	return parsed, err
}
