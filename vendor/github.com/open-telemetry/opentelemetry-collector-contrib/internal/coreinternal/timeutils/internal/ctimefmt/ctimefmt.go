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
	"time"
)

var (
	ctimeRegexp                      = regexp.MustCompile(`%.`)
	invalidFractionalSecondsStrptime = regexp.MustCompile(`[^.,]%[Lfs]`)
	decimalsRegexp                   = regexp.MustCompile(`\d`)
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
	"%s": "99999999",
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
//	%s - Nanosecond as a decimal number, zero-padded on the left (00000000, ..., 99999999)
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
	native, err := ToNative(format)
	if err != nil {
		return "", err
	}
	return t.Format(native), nil
}

// Parse parses a ctime-like formatted string (e.g. "%Y-%m-%d ...") and returns
// the time value it represents.
//
// Refer to Format() function documentation for possible directives.
func Parse(format, value string) (time.Time, error) {
	native, err := ToNative(format)
	if err != nil {
		return time.Time{}, nil
	}
	return time.Parse(native, value)
}

// ToNative converts ctime-like format string to Go native layout
// (which is used by time.Time.Format() and time.Parse() functions).
func ToNative(format string) (string, error) {
	var errs []error
	replaceFunc := func(directive string) string {
		if subst, ok := ctimeSubstitutes[directive]; ok {
			return subst
		}
		errs = append(errs, errors.New("unsupported ctimefmt.ToNative() directive: "+directive))
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
			errs = append(errs, errors.New("unsupported ctimefmt.ToNative() directive: "+directive))
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf("invalid strptime format: %v", errs)
	}
	return nil
}

// GetNativeSubstitutes analyzes the provided format string and returns a map where each
// key is a Go native layout element (as used in time.Format) found in the format, and
// each value is the corresponding ctime-like directive.
func GetNativeSubstitutes(format string) map[string]string {
	nativeDirectives := map[string]string{}
	directives := ctimeRegexp.FindAllString(format, -1)
	for _, directive := range directives {
		if val, ok := ctimeSubstitutes[directive]; ok {
			nativeDirectives[val] = directive
		}
	}
	return nativeDirectives
}
