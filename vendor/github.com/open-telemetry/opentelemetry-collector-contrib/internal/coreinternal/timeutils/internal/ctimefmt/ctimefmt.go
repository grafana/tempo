// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Keep the original license.

// Copyright 2019 Dmitry A. Mottl. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.

package ctimefmt // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils/internal/ctimefmt"

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/elastic/lunes"
)

var (
	ctimeRegexp                      = regexp.MustCompile(`%.`)
	invalidFractionalSecondsStrptime = regexp.MustCompile(`[^.,]%[Lfs]`)
	decimalsRegexp                   = regexp.MustCompile(`\d`)
	leadingSpaceRegexp               = regexp.MustCompile(`^\s+`)
)

var ctimeSubstitutes = map[string]string{
	"%Y": "2006",
	"%y": "06",
	"%m": "01",
	"%o": "_1",
	"%q": "1",
	"%b": "Jan",
	"%h": "Jan",
	"%B": "January",
	"%d": "02",
	"%e": "_2",
	"%g": "2",
	"%a": "Mon",
	"%A": "Monday",
	"%H": "15",
	"%l": "3",
	"%I": "03",
	"%p": "PM",
	"%P": "pm",
	"%M": "04",
	"%S": "05",
	"%L": "999",
	"%f": "999999",
	"%s": "999999999",
	"%Z": "MST",
	"%z": "Z0700",
	"%w": "-070000",
	"%i": "-07",
	"%j": "-07:00",
	"%k": "-07:00:00",
	"%D": "01/02/2006",
	"%x": "01/02/2006",
	"%F": "2006-01-02",
	"%T": "15:04:05",
	"%X": "15:04:05",
	"%r": "03:04:05 pm",
	"%R": "15:04",
	"%n": "\n",
	"%t": "\t",
	"%%": "%",
	"%c": "Mon Jan 02 15:04:05 2006",
}

// Format returns a textual representation of the time value formatted
// according to ctime-like format string. Possible directives are:
//
//	%Y - Year, zero-padded (0001, 0002, ..., 2019, 2020, ..., 9999)
//	%y - Year, last two digits, zero-padded (01, ..., 99)
//	%m - Month as a decimal number (01, 02, ..., 12)
//	%o - Month as a space-padded number ( 1, 2, ..., 12)
//	%q - Month as a unpadded number (1,2,...,12)
//	%b, %h - Abbreviated month name (Jan, Feb, ...)
//	%B - Full month name (January, February, ...)
//	%d - Day of the month, zero-padded (01, 02, ..., 31)
//	%e - Day of the month, space-padded ( 1, 2, ..., 31)
//	%g - Day of the month, unpadded (1,2,...,31)
//	%a - Abbreviated weekday name (Sun, Mon, ...)
//	%A - Full weekday name (Sunday, Monday, ...)
//	%H - Hour (24-hour clock) as a zero-padded decimal number (00, ..., 24)
//	%I - Hour (12-hour clock) as a zero-padded decimal number (00, ..., 12)
//	%l - Hour (12-hour clock: 0, ..., 12)
//	%p - Locale’s equivalent of either AM or PM
//	%P - Locale’s equivalent of either am or pm
//	%M - Minute, zero-padded (00, 01, ..., 59)
//	%S - Second as a zero-padded decimal number (00, 01, ..., 59)
//	%L - Millisecond as a decimal number, zero-padded on the left (000, 001, ..., 999)
//	%f - Microsecond as a decimal number, zero-padded on the left (000000, ..., 999999)
//	%s - Nanosecond as a decimal number, zero-padded on the left (000000000, ..., 999999999)
//	%z - UTC offset in the form ±HHMM[SS[.ffffff]] or empty(+0000, -0400)
//	%Z - Timezone name or abbreviation or empty (UTC, EST, CST)
//	%D, %x - Short MM/DD/YYYY date, equivalent to %m/%d/%y
//	%F - Short YYYY-MM-DD date, equivalent to %Y-%m-%d
//	%T, %X - ISO 8601 time format (HH:MM:SS), equivalent to %H:%M:%S
//	%r - 12-hour clock time (02:55:02 pm)
//	%R - 24-hour HH:MM time, equivalent to %H:%M
//	%n - New-line character ('\n')
//	%t - Horizontal-tab character ('\t')
//	%% - A % sign
//	%c - Date and time representation (Mon Jan 02 15:04:05 2006)
func Format(format string, t time.Time) (string, error) {
	native, err := toNative(format)
	if err != nil {
		return "", err
	}
	return t.Format(native), nil
}

type ParseFunc func(layout string) (time.Time, error)

// Parse parses a ctime-like formatted string (e.g. "%Y-%m-%d ...")
// and returns the time value it represents and the Go layout string
// that successfully parsed the input.
//
// It differs from Format in that it will attempt to parse the string
// multiple times in order to handle format specifiers that can have
// multiple valid formats such as %z. Notable differences include:
//
// - Leading whitespace before a digit is ignored
// - Numbers may or may not have leading zero digits
// - Multiple time zone formats are supported
//
// Refer to strptime(3) and Format() function documentation for possible directives.
//
// Note: The returned ParseError will indicate the ctime directive that failed to parse.
func Parse(format string, parse ParseFunc) (time.Time, string, error) {
	indexes := ctimeRegexp.FindAllStringIndex(format, -1)
	t, layout, err := iterativeParse("", format, 0, indexes, parse)
	if timeErr, ok := errors.AsType[*time.ParseError](err); ok {
		timeErr.Layout = format
	}
	return t, layout, err
}

// Alternative formats that allow more flexible ctime-compatible parsing.
// The values are split into discrete time.Parse elements so they can be identified in ParseError.LayoutElem.
var ctimeParseSubstitutes = map[string][][]string{
	"%m": {{"1"}},
	"%o": {{"1"}},
	"%q": {{"1"}},
	"%d": {{"2"}},
	"%e": {{"2"}},
	"%g": {{"2"}},
	"%I": {{"3"}},
	"%M": {{"4"}},
	"%S": {{"5"}},
	"%D": {
		// N.B. The docs above say that %D is equivalent to %m/%d/%y, but the implementation of Format uses %m/%d/%Y. We try to parse as both.
		{"1", "/", "2", "/", "2006"},
		{"1", "/", "2", "/", "06"},
	},
	"%x": {
		{"1", "/", "2", "/", "2006"},
		{"1", "/", "2", "/", "06"},
	},
	"%F": {{"2006", "-", "1", "-", "2"}},
	"%T": {{"15", ":", "4", ":", "5"}},
	"%X": {{"15", ":", "4", ":", "5"}},
	"%r": {{"3", ":", "4", ":", "5", " ", "pm"}},
	"%R": {{"15", ":", "4"}},
	"%c": {{"Mon", " ", "Jan", " ", "2", " ", "15", ":", "4", ":", "5", " ", "2006"}},
}

func iterativeParse(partialLayout, format string, startIndex int, indexes [][]int, parse ParseFunc) (out time.Time, layout string, err error) {
	if len(indexes) == 0 {
		layout = partialLayout + format[startIndex:]
		out, err = parse(layout)
		return out, layout, err
	}
	partialLayout += format[startIndex:indexes[0][0]]
	directive := format[indexes[0][0]:indexes[0][1]]
	// tryElements will attempt to match the time with elements.
	// It returns the index of the failing element and the remaining unparsed input, or -1 if none of the elements were responsible for a failure.
	tryElements := func(elements ...string) (int, string) {
		out, layout, err = iterativeParse(partialLayout+strings.Join(elements, ""), format, indexes[0][1], indexes[1:], parse)
		var timeErr *time.ParseError
		if errors.As(err, &timeErr) && timeErr.LayoutElem != "" {
			if i := slices.Index(elements, timeErr.LayoutElem); i >= 0 {
				timeErr.LayoutElem = directive
				return i, timeErr.ValueElem
			}
		}
		var lunesErr *lunes.ErrLayoutMismatch
		if errors.As(err, &lunesErr) && lunesErr.LayoutElem != "" {
			if i := slices.Index(elements, lunesErr.LayoutElem); i >= 0 {
				lunesErr.LayoutElem = directive
				return i, "unknown"
			}
		}
		return -1, ""
	}
	// try will attempt to match the time with elements and optional leading spaces
	try := func(elements ...string) (int, string) {
		for {
			i, remainder := tryElements(elements...)
			if i >= 0 {
				switch elements[i][0] {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'Z':
					space := leadingSpaceRegexp.FindString(remainder)
					if space != "" {
						elements = slices.Insert(append([]string{}, elements...), i, space)
						continue
					}
				}
			}
			return i, remainder
		}
	}
	if directive == "%z" {
		if i, _ := try("Z0700"); i < 0 {
			return out, layout, err
		}
		if i, _ := try("Z07:00"); i < 0 {
			return out, layout, err
		}
		if i, _ := try("Z07"); i < 0 {
			return out, layout, err
		}
		return out, layout, err
	}
	if substs, ok := ctimeParseSubstitutes[directive]; ok {
		for _, subst := range substs {
			if i, _ := try(subst...); i < 0 {
				break
			}
		}
		// Don't fall back to the original substitutes, or we will generate incorrect error messages.
		return out, layout, err
	}
	if subst, ok := ctimeSubstitutes[directive]; ok {
		try(subst)
		return out, layout, err
	}
	return time.Time{}, "", fmt.Errorf("unsupported ctimefmt.FlexibleParse() directive: %s", directive)
}

// toNative converts ctime-like format string to Go native layout
// (which is used by time.Time.Format() and time.Parse() functions).
func toNative(format string) (string, error) {
	var errs []error
	replaceFunc := func(directive string) string {
		if subst, ok := ctimeSubstitutes[directive]; ok {
			return subst
		}
		errs = append(errs, errors.New("unsupported ctimefmt.toNative() directive: "+directive))
		return ""
	}

	replaced := ctimeRegexp.ReplaceAllStringFunc(format, replaceFunc)
	if len(errs) != 0 {
		return "", fmt.Errorf("convert to go time format: %v", errs)
	}

	return replaced, nil
}

func Validate(format string) error {
	if match := decimalsRegexp.FindString(format); match != "" {
		return errors.New("format string should not contain decimals")
	}

	if match := invalidFractionalSecondsStrptime.FindString(format); match != "" {
		return fmt.Errorf("invalid fractional seconds directive: '%s'. must be preceded with '.' or ','", match)
	}

	directives := ctimeRegexp.FindAllString(format, -1)

	var errs []error
	for _, directive := range directives {
		if _, ok := ctimeSubstitutes[directive]; !ok {
			errs = append(errs, errors.New("unsupported ctimefmt.toNative() directive: "+directive))
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf("invalid strptime format: %v", errs)
	}
	return nil
}
