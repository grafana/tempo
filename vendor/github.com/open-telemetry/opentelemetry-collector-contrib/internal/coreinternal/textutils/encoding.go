// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package textutils // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/textutils"

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
)

var encodingOverrides = map[string]encoding.Encoding{
	"utf-16":    unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf16":     unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf-8":     unicode.UTF8,
	"utf8":      unicode.UTF8,
	"utf-8-raw": UTF8Raw,
	"utf8-raw":  UTF8Raw,
	"ascii":     unicode.UTF8,
	"us-ascii":  unicode.UTF8,
	"nop":       encoding.Nop,
	"":          unicode.UTF8,
}

func LookupEncoding(enc string) (encoding.Encoding, error) {
	if e, ok := encodingOverrides[strings.ToLower(enc)]; ok {
		return e, nil
	}
	e, err := ianaindex.IANA.Encoding(enc)
	if err != nil {
		return nil, fmt.Errorf("unsupported encoding '%s'", enc)
	}
	if e == nil {
		return nil, fmt.Errorf("no charmap defined for encoding '%s'", enc)
	}
	return e, nil
}

func IsNop(enc string) bool {
	e, err := LookupEncoding(enc)
	if err != nil {
		return false
	}
	return e == encoding.Nop
}

// DecodeAsString converts the given encoded bytes using the given decoder. It returns the converted
// bytes or nil, err if any error occurred.
func DecodeAsString(decoder *encoding.Decoder, buf []byte) (string, error) {
	dstBuf, err := decoder.Bytes(buf)
	if err != nil {
		return "", err
	}
	return UnsafeBytesAsString(dstBuf), nil
}

// UnsafeBytesAsString converts the byte array to string.
// This function must be called iff the input buffer is not going to be re-used after.
func UnsafeBytesAsString(buf []byte) string {
	return unsafe.String(unsafe.SliceData(buf), len(buf))
}
