package archives

import (
	"context"
	"io"
	"strings"

	"github.com/klauspost/compress/zlib"
)

func init() {
	RegisterFormat(Zlib{})
}

// Zlib facilitates zlib compression.
type Zlib struct {
	CompressionLevel int
}

func (Zlib) Extension() string { return ".zz" }
func (Zlib) MediaType() string { return "application/zlib" }

func (zz Zlib) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if strings.Contains(strings.ToLower(filename), zz.Extension()) {
		mr.ByName = true
	}

	// match file header
	buf, err := readAtMost(stream, 2)
	// If an error occurred or buf is not 2 bytes we can't check the header
	if err != nil || len(buf) < 2 {
		return mr, err
	}

	mr.ByStream = isValidZlibHeader(buf[0], buf[1])

	return mr, nil
}

func (zz Zlib) OpenWriter(w io.Writer) (io.WriteCloser, error) {
	level := zz.CompressionLevel
	if level == 0 {
		level = zlib.DefaultCompression
	}
	return zlib.NewWriterLevel(w, level)
}

func (Zlib) OpenReader(r io.Reader) (io.ReadCloser, error) {
	return zlib.NewReader(r)
}

func isValidZlibHeader(first, second byte) bool {
	// Define all 32 valid zlib headers, see https://stackoverflow.com/questions/9050260/what-does-a-zlib-header-look-like/54915442#54915442
	validHeaders := map[uint16]struct{}{
		0x081D: {}, 0x085B: {}, 0x0899: {}, 0x08D7: {},
		0x1819: {}, 0x1857: {}, 0x1895: {}, 0x18D3: {},
		0x2815: {}, 0x2853: {}, 0x2891: {}, 0x28CF: {},
		0x3811: {}, 0x384F: {}, 0x388D: {}, 0x38CB: {},
		0x480D: {}, 0x484B: {}, 0x4889: {}, 0x48C7: {},
		0x5809: {}, 0x5847: {}, 0x5885: {}, 0x58C3: {},
		0x6805: {}, 0x6843: {}, 0x6881: {}, 0x68DE: {},
		0x7801: {}, 0x785E: {}, 0x789C: {}, 0x78DA: {},
	}

	// Combine the first and second bytes into a single 16-bit, big-endian value
	header := uint16(first)<<8 | uint16(second)

	// Check if the header is in the map of valid headers
	_, isValid := validHeaders[header]
	return isValid
}
