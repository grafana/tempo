// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampling // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"

import (
	"errors"
	"io"
	"strings"

	"go.uber.org/multierr"
)

// KV represents a key-value parsed from a section of the TraceState.
type KV struct {
	Key   string
	Value string
}

// ErrTraceStateSize is returned when a TraceState is over its
// size limit, as specified by W3C.
var ErrTraceStateSize = errors.New("invalid tracestate size")

// keyValueScanner defines distinct scanner behaviors for lists of
// key-values.
type keyValueScanner struct {
	// maxItems is 32 or -1
	maxItems int
	// trim is set if OWS (optional whitespace) should be removed
	trim bool
	// separator is , or ;
	separator byte
	// equality is = or :
	equality byte
}

// commonTraceState is embedded in both W3C and OTel trace states.
type commonTraceState struct {
	kvs []KV
}

// ExtraValues returns additional values are carried in this
// tracestate object (W3C or OpenTelemetry).
func (cts commonTraceState) ExtraValues() []KV {
	return cts.kvs
}

// trimOws removes optional whitespace on both ends of a string.
// this uses the strict definition for optional whitespace tiven
// in https://www.w3.org/TR/trace-context/#tracestate-header-field-values
func trimOws(input string) string {
	return strings.Trim(input, " \t")
}

// scanKeyValues is common code to scan either W3C or OTel tracestate
// entries, as parameterized in the keyValueScanner struct.
func (s keyValueScanner) scanKeyValues(input string, f func(key, value string) error) error {
	var rval error
	items := 0
	for input != "" {
		items++
		if s.maxItems > 0 && items >= s.maxItems {
			// W3C specifies max 32 entries, tested here
			// instead of via the regexp.
			return ErrTraceStateSize
		}

		sep := strings.IndexByte(input, s.separator)

		var member string
		if sep < 0 {
			member = input
			input = ""
		} else {
			member = input[:sep]
			input = input[sep+1:]
		}

		if s.trim {
			// Trim only required for W3C; OTel does not
			// specify whitespace for its value encoding.
			member = trimOws(member)
		}

		if member == "" {
			// W3C allows empty list members.
			continue
		}

		eq := strings.IndexByte(member, s.equality)
		if eq < 0 {
			// We expect to find the `s.equality`
			// character in this string because we have
			// already validated the whole input syntax
			// before calling this parser.  I.e., this can
			// never happen, and if it did, the result
			// would be to skip malformed entries.
			continue
		}
		if err := f(member[:eq], member[eq+1:]); err != nil {
			rval = multierr.Append(rval, err)
		}
	}
	return rval
}

// serializer assists with checking and combining errors from
// (io.StringWriter).WriteString().
type serializer struct {
	writer io.StringWriter
	err    error
}

// write handles errors from io.StringWriter.
func (ser *serializer) write(str string) {
	_, err := ser.writer.WriteString(str)
	ser.check(err)
}

// check handles errors (e.g., from another serializer).
func (ser *serializer) check(err error) {
	ser.err = multierr.Append(ser.err, err)
}

// =============================================================================
// Character validation functions shared by W3C and OTel tracestate parsers.
// These hand-written validators are significantly faster than regex-based
// validation (30-60x speedup).
// =============================================================================

// isLcAlpha returns true if c is a lowercase ASCII letter (a-z).
func isLcAlpha(c byte) bool {
	return c >= 'a' && c <= 'z'
}

// isLcAlphaNum returns true if c is a lowercase ASCII letter or digit.
func isLcAlphaNum(c byte) bool {
	return isLcAlpha(c) || (c >= '0' && c <= '9')
}

// isValidKeyChar returns true if c is valid in a W3C tracestate key
// (lowercase alphanumeric or one of: _ - * /).
func isValidKeyChar(c byte) bool {
	return isLcAlphaNum(c) || c == '_' || c == '-' || c == '*' || c == '/'
}

// isValidKeyChars returns true if all characters in s are valid W3C key characters.
func isValidKeyChars(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isValidKeyChar(s[i]) {
			return false
		}
	}
	return true
}
