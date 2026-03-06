package archives

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
)

func init() {
	RegisterFormat(Zstd{})
}

// Zstd facilitates Zstandard compression.
type Zstd struct {
	EncoderOptions []zstd.EOption
	DecoderOptions []zstd.DOption
}

func (Zstd) Extension() string { return ".zst" }
func (Zstd) MediaType() string { return "application/zstd" }

func (zs Zstd) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if strings.Contains(strings.ToLower(filename), zs.Extension()) {
		mr.ByName = true
	}

	// match file header
	buf, err := readAtMost(stream, len(zstdHeader))
	if err != nil {
		return mr, err
	}
	mr.ByStream = bytes.Equal(buf, zstdHeader)

	return mr, nil
}

func (zs Zstd) OpenWriter(w io.Writer) (io.WriteCloser, error) {
	return zstd.NewWriter(w, zs.EncoderOptions...)
}

func (zs Zstd) OpenReader(r io.Reader) (io.ReadCloser, error) {
	zr, err := zstd.NewReader(r, zs.DecoderOptions...)
	if err != nil {
		return nil, err
	}
	return errorCloser{zr}, nil
}

type errorCloser struct {
	*zstd.Decoder
}

func (ec errorCloser) Close() error {
	ec.Decoder.Close()
	return nil
}

// magic number at the beginning of Zstandard files
// https://github.com/facebook/zstd/blob/6211bfee5ec24dc825c11751c33aa31d618b5f10/doc/zstd_compression_format.md
var zstdHeader = []byte{0x28, 0xb5, 0x2f, 0xfd}
