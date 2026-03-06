package archives

import (
	"bytes"
	"context"
	"io"
	"strings"

	fastxz "github.com/therootcompany/xz"
	"github.com/ulikunitz/xz"
)

func init() {
	RegisterFormat(Xz{})
}

// Xz facilitates xz compression.
type Xz struct{}

func (Xz) Extension() string { return ".xz" }
func (Xz) MediaType() string { return "application/x-xz" }

func (x Xz) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if strings.Contains(strings.ToLower(filename), x.Extension()) {
		mr.ByName = true
	}

	// match file header
	buf, err := readAtMost(stream, len(xzHeader))
	if err != nil {
		return mr, err
	}
	mr.ByStream = bytes.Equal(buf, xzHeader)

	return mr, nil
}

func (Xz) OpenWriter(w io.Writer) (io.WriteCloser, error) {
	return xz.NewWriter(w)
}

func (Xz) OpenReader(r io.Reader) (io.ReadCloser, error) {
	xr, err := fastxz.NewReader(r, 0)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(xr), err
}

// magic number at the beginning of xz files; see section 2.1.1.1
// of https://tukaani.org/xz/xz-file-format.txt
var xzHeader = []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}
