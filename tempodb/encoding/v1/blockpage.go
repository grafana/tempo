package v1

import (
	"errors"
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

var supportedEncoding = []Encoding{
	EncNone,
	EncGZIP,
	EncLZ4_64k,
	EncLZ4_256k,
	EncLZ4_1M,
	EncLZ4_4M,
	EncSnappy,
}

const maxEncoding = EncSnappy

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
	for _, e := range supportedEncoding {
		if strings.EqualFold(e.String(), enc) {
			return e, nil
		}
	}
	return 0, fmt.Errorf("invalid encoding: %s, supported: %s", enc, SupportedEncoding())
}

// SupportedEncoding returns the list of supported Encoding.
func SupportedEncoding() string {
	var sb strings.Builder
	for i := range supportedEncoding {
		sb.WriteString(supportedEncoding[i].String())
		if i != len(supportedEncoding)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String()
}

// BlockPage represents a page in a block.  This is a collection of contiguous traces in the v0 format
// proceeded by an encoding byte.  Note that block pages are written in the buffered appender.
type BlockPage struct {
	encoding     Encoding
	encodedBytes []byte
}

// NewBlockPage returns a BlockPage given the page bytes
func NewBlockPage(buf []byte) (*BlockPage, error) {
	if len(buf) == 0 {
		return nil, errors.New("can't create a 0 length block page")
	}

	// the first byte should be the encoding
	if buf[0] > byte(maxEncoding) {
		return nil, fmt.Errorf("unknown encoding %d", buf[0])
	}

	encoding := Encoding(buf[0])
	return &BlockPage{
		encoding:     encoding,
		encodedBytes: buf[1:],
	}, nil
}
