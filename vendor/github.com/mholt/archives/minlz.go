package archives

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/minio/minlz"
)

func init() {
	RegisterFormat(MinLZ{})
}

// MinLZ facilitates MinLZ compression. See
// https://github.com/minio/minlz/blob/main/SPEC.md
// and
// https://blog.min.io/minlz-compression-algorithm/.
type MinLZ struct{}

func (MinLZ) Extension() string { return ".mz" }
func (MinLZ) MediaType() string { return "application/x-minlz-compressed" }

func (mz MinLZ) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if filepath.Ext(strings.ToLower(filename)) == ".mz" {
		mr.ByName = true
	}

	// match file header
	buf, err := readAtMost(stream, len(mzHeader))
	if err != nil {
		return mr, err
	}
	mr.ByStream = bytes.Equal(buf, mzHeader)

	return mr, nil
}

func (MinLZ) OpenWriter(w io.Writer) (io.WriteCloser, error) {
	return minlz.NewWriter(w), nil
}

func (MinLZ) OpenReader(r io.Reader) (io.ReadCloser, error) {
	mr := minlz.NewReader(r)
	return io.NopCloser(mr), nil
}

var mzHeader = []byte("\xff\x06\x00\x00MinLz")
