// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build cgo && unix && !aix

// This file is ONLY used as part of TestTimeParserStrptimeCgo to verify that TestParseStrptime is correct.

package timeutils // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/timeutils"

// #define _XOPEN_SOURCE
// // _DEFAULT_SOURCE only exists since glibc 2.19 (2014); _BSD_SOURCE is the
// // spelling older glibc versions understand for the same feature set. Both
// // are defined so tm_gmtoff is visible regardless of glibc version.
// #define _DEFAULT_SOURCE
// #define _BSD_SOURCE
// #include <stdlib.h>
// #include <time.h>
//
// // glibc's strptime only started accepting "Z" and colon-separated
// // ([+-]HH:MM) UTC offsets in %z as of glibc 2.23 (2016). Musl rejects them too.
// #if defined(__GLIBC__) && (__GLIBC__ > 2 || (__GLIBC__ == 2 && __GLIBC_MINOR__ >= 23))
// #define STRPTIME_SUPPORTS_EXTENDED_TZ 1
// #else
// #define STRPTIME_SUPPORTS_EXTENDED_TZ 0
// #endif
import (
	"C" //nolint: gocritic // Buggy: https://github.com/go-critic/go-critic/issues/845
)

import (
	"errors"
	"fmt"
	"time"
	"unsafe" //nolint: gocritic
)

// SupportsExtendedTZOffset reports whether the host libc's strptime accepts
// the "Z" and colon-separated ([+-]HH:MM) forms of %z.
const SupportsExtendedTZOffset = C.STRPTIME_SUPPORTS_EXTENDED_TZ != 0

func tm2Time(tm C.struct_tm) time.Time {
	return time.Date(
		int(tm.tm_year+1900),
		time.Month(tm.tm_mon+1),
		int(tm.tm_mday),
		int(tm.tm_hour),
		int(tm.tm_min),
		int(tm.tm_sec),
		0,
		time.FixedZone((time.Duration(tm.tm_gmtoff)*time.Second).String(), int(tm.tm_gmtoff)),
	)
}

// Wrap libc's strptime for use in TestTimeParserStrptimeCgo
func CStrptime(s, format string) (time.Time, error) {
	cformat := C.CString(format)
	defer C.free(unsafe.Pointer(cformat))

	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))

	tm := C.struct_tm{
		// Default time zone must match the TestParseStrptime
		tm_gmtoff: -7 * 3600,
	}

	out := C.strptime(cs, cformat, &tm) //nolint: gocritic // Buggy: https://github.com/go-critic/go-critic/issues/897

	t := tm2Time(tm)

	if out == nil {
		return t, errors.New("strptime failed to parse the whole string")
	} else if unsafe.Pointer(out) != unsafe.Add(unsafe.Pointer(cs), len(s)) {
		return t, fmt.Errorf("strptime failed to parse: remainder %q", C.GoString(out))
	}
	return t, nil
}
