package archives

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/sorairolake/lzip-go"
)

func init() {
	RegisterFormat(Lzip{})
}

// Lzip facilitates lzip compression.
type Lzip struct{}

func (Lzip) Extension() string { return ".lz" }
func (Lzip) MediaType() string { return "application/x-lzip" }

func (lz Lzip) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if filepath.Ext(strings.ToLower(filename)) == lz.Extension() {
		mr.ByName = true
	}

	// match file header
	buf, err := readAtMost(stream, len(lzipHeader))
	if err != nil {
		return mr, err
	}
	mr.ByStream = bytes.Equal(buf, lzipHeader)

	return mr, nil
}

func (Lzip) OpenWriter(w io.Writer) (io.WriteCloser, error) {
	return lzip.NewWriter(w), nil
}

func (Lzip) OpenReader(r io.Reader) (io.ReadCloser, error) {
	lzr, err := lzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(lzr), err
}

// magic number at the beginning of lzip files
// https://datatracker.ietf.org/doc/html/draft-diaz-lzip-09#section-2
var lzipHeader = []byte("LZIP")
