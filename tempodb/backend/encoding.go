package backend

import (
	"fmt"
	"strings"
)

// Encoding is the identifier for a chunk encoding.
type Encoding byte

// The different available encodings.
// Make sure to preserve the order, as these numeric values are written to the chunks!
const (
	EncNone Encoding = iota
	EncGZIP
	EncLZ4_64k
	EncLZ4_256k
	EncLZ4_1M
	EncLZ4_4M
	EncSnappy
)

// SupportedEncoding is a slice of all supported encodings
var SupportedEncoding = []Encoding{
	EncNone,
	EncGZIP,
	EncLZ4_64k,
	EncLZ4_256k,
	EncLZ4_1M,
	EncLZ4_4M,
	EncSnappy,
}

func (e Encoding) String() string {
	switch e {
	case EncNone:
		return "none"
	case EncGZIP:
		return "gzip"
	case EncLZ4_64k:
		return "lz4-64k"
	case EncLZ4_256k:
		return "lz4-256k"
	case EncLZ4_1M:
		return "lz4-1M"
	case EncLZ4_4M:
		return "lz4"
	case EncSnappy:
		return "snappy"
	default:
		return "unsupported"
	}
}

// ParseEncoding parses an chunk encoding (compression algorithm) by its name.
func ParseEncoding(enc string) (Encoding, error) {
	for _, e := range SupportedEncoding {
		if strings.EqualFold(e.String(), enc) {
			return e, nil
		}
	}
	return 0, fmt.Errorf("invalid encoding: %s, supported: %s", enc, SupportedEncodingString())
}

// SupportedEncodingString returns the list of supported Encoding.
func SupportedEncodingString() string {
	var sb strings.Builder
	for i := range SupportedEncoding {
		sb.WriteString(SupportedEncoding[i].String())
		if i != len(SupportedEncoding)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String()
}
