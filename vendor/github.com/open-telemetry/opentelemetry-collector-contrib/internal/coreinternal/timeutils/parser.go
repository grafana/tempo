// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package timeutils // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils"

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/lunes"

	strptime "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils/internal/ctimefmt"
)

var invalidFractionalSecondsGoTime = regexp.MustCompile(`[^.,9]9+`)

func StrptimeToGotime(layout string) (string, error) {
	return strptime.ToNative(layout)
}

func ParseStrptime(layout string, value any, location *time.Location) (time.Time, error) {
	goLayout, err := strptime.ToNative(layout)
	if err != nil {
		return time.Time{}, err
	}
	return ParseGotime(goLayout, value, location)
}

// ParseLocalizedStrptime is like ParseLocalizedGotime, but instead of using the native Go time layout,
// it uses the ctime-like format.
func ParseLocalizedStrptime(layout string, value any, location *time.Location, language string) (time.Time, error) {
	goLayout, err := strptime.ToNative(layout)
	if err != nil {
		return time.Time{}, err
	}

	return ParseLocalizedGotime(goLayout, value, location, language)
}

func GetLocation(location *string, layout *string) (*time.Location, error) {
	if location != nil && *location != "" {
		// If location is specified, it must be in the local timezone database
		loc, err := time.LoadLocation(*location)
		if err != nil {
			return nil, fmt.Errorf("failed to load location %s: %w", *location, err)
		}
		return loc, nil
	}

	if layout != nil && strings.HasSuffix(*layout, "Z") {
		// If a timestamp ends with 'Z', it should be interpreted at Zulu (UTC) time
		return time.UTC, nil
	}

	return time.Local, nil
}

// ParseLocalizedGotime is like ParseGotime, but instead of parsing a formatted time in
// English, it parses a value in foreign language, and returns the [time.Time] it represents.
// The language argument must be a well-formed BCP 47 language tag (e.g.: "en", "en-US"), and
// a known CLDR locale.
func ParseLocalizedGotime(layout string, value any, location *time.Location, language string) (time.Time, error) {
	stringValue, err := convertParsingValue(value)
	if err != nil {
		return time.Time{}, err
	}

	translatedVal, err := lunes.Translate(layout, stringValue, language)
	if err != nil {
		return time.Time{}, err
	}

	return ParseGotime(layout, translatedVal, location)
}

func ParseGotime(layout string, value any, location *time.Location) (time.Time, error) {
	timeValue, err := parseGotime(layout, value, location)
	if err != nil {
		return time.Time{}, err
	}
	return SetTimestampYear(timeValue), nil
}

func parseGotime(layout string, value any, location *time.Location) (time.Time, error) {
	str, err := convertParsingValue(value)
	if err != nil {
		return time.Time{}, err
	}

	result, err := time.ParseInLocation(layout, str, location)

	// Depending on the timezone database, we may get a pseudo-matching timezone
	// This is apparent when the zone is not "UTC", but the offset is still 0
	zone, offset := result.Zone()
	if offset != 0 || zone == "UTC" {
		return result, err
	}

	// Manually look up the location based on the zone
	loc, locErr := time.LoadLocation(zone)
	if locErr != nil {
		// can't correct offset, just return what we have
		return result, fmt.Errorf("failed to load location %s: %w", zone, locErr)
	}

	// Reparse the timestamp, with the location
	resultLoc, locErr := time.ParseInLocation(layout, str, loc)
	if locErr != nil {
		// can't correct offset, just return original result
		return result, err
	}

	return resultLoc, locErr
}

func convertParsingValue(value any) (string, error) {
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return "", fmt.Errorf("type %T cannot be parsed as a time", value)
	}

	return str, nil
}

// SetTimestampYear sets the year of a timestamp to the current year.
// This is needed because year is missing from some time formats, such as rfc3164.
func SetTimestampYear(t time.Time) time.Time {
	if t.Year() > 0 {
		return t
	}
	n := Now()
	d := time.Date(n.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	// Assume the timestamp is from last year if its month and day are
	// more than 7 days past the current date.
	// i.e. If today is January 1, but the timestamp is February 1, it's safe
	// to assume the timestamp is from last year.
	if d.After(n.AddDate(0, 0, 7)) {
		d = d.AddDate(-1, 0, 0)
	}
	return d
}

// ValidateStrptime checks the given strptime layout and returns an error if it detects any known issues
// that prevent it from being parsed.
func ValidateStrptime(layout string) error {
	return strptime.Validate(layout)
}

func ValidateGotime(layout string) error {
	if match := invalidFractionalSecondsGoTime.FindString(layout); match != "" {
		return fmt.Errorf("invalid fractional seconds directive: '%s'. must be preceded with '.' or ','", match)
	}

	return nil
}

// ValidateLocale checks the given locale and returns an error if the language tag
// is not supported by the localized parser functions.
func ValidateLocale(locale string) error {
	_, err := lunes.NewDefaultLocale(locale)
	if err == nil {
		return nil
	}

	var e *lunes.ErrUnsupportedLocale
	if errors.As(err, &e) {
		return fmt.Errorf("unsupported locale '%s', value must be a supported BCP 47 language tag", locale)
	}

	return fmt.Errorf("invalid locale '%s': %w", locale, err)
}

// GetStrptimeNativeSubstitutes analyzes the provided format string and returns a map
// where each key is a Go native layout element (as used in time.Format) found in the
// format, and each value is the corresponding ctime-like directive.
func GetStrptimeNativeSubstitutes(format string) map[string]string {
	return strptime.GetNativeSubstitutes(format)
}

type strptimeParseErr struct {
	err               *time.ParseError
	ctimeLayout       string
	nativeSubstitutes map[string]string
}

func (e *strptimeParseErr) Error() string {
	if e.err.Message == "" {
		layoutElem, ok := e.nativeSubstitutes[e.err.LayoutElem]
		if !ok {
			layoutElem = e.err.LayoutElem
		}
		return "parsing time " +
			strconv.Quote(e.err.Value) + " as " +
			strconv.Quote(e.ctimeLayout) + ": cannot parse " +
			strconv.Quote(e.err.ValueElem) + " as " +
			strconv.Quote(layoutElem)
	}
	return "parsing time " + strconv.Quote(e.err.Value) + e.err.Message
}

func ToStrptimeParseError(err *time.ParseError, ctimeLayout string, nativeSubstitutes map[string]string) error {
	return &strptimeParseErr{err, ctimeLayout, nativeSubstitutes}
}

// Allows tests to override with deterministic value
var Now = time.Now
