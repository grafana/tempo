package archives

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/pierrec/lz4/v4"
)

func init() {
	RegisterFormat(Lz4{})
}

// Lz4 facilitates LZ4 compression.
type Lz4 struct {
	CompressionLevel int
}

func (Lz4) Extension() string { return ".lz4" }
func (Lz4) MediaType() string { return "application/x-lz4" }

func (lz Lz4) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if strings.Contains(strings.ToLower(filename), lz.Extension()) {
		mr.ByName = true
	}

	// match file header
	buf, err := readAtMost(stream, len(lz4Header))
	if err != nil {
		return mr, err
	}
	mr.ByStream = bytes.Equal(buf, lz4Header)

	return mr, nil
}

func (lz Lz4) OpenWriter(w io.Writer) (io.WriteCloser, error) {
	lzw := lz4.NewWriter(w)
	options := []lz4.Option{
		lz4.CompressionLevelOption(lz4.CompressionLevel(lz.CompressionLevel)),
	}
	if err := lzw.Apply(options...); err != nil {
		return nil, err
	}
	return lzw, nil
}

func (Lz4) OpenReader(r io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(lz4.NewReader(r)), nil
}

var lz4Header = []byte{0x04, 0x22, 0x4d, 0x18}
