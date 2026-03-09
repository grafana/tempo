package sevenzip

import (
	"errors"
	"io"
	"sync"

	"github.com/bodgit/sevenzip/internal/aes7z"
	"github.com/bodgit/sevenzip/internal/bcj2"
	"github.com/bodgit/sevenzip/internal/bra"
	"github.com/bodgit/sevenzip/internal/brotli"
	"github.com/bodgit/sevenzip/internal/bzip2"
	"github.com/bodgit/sevenzip/internal/deflate"
	"github.com/bodgit/sevenzip/internal/delta"
	"github.com/bodgit/sevenzip/internal/lz4"
	"github.com/bodgit/sevenzip/internal/lzma"
	"github.com/bodgit/sevenzip/internal/lzma2"
	"github.com/bodgit/sevenzip/internal/zstd"
)

// Decompressor describes the function signature that decompression/decryption
// methods must implement to return a new instance of themselves. They are
// passed any property bytes, the size of the stream and a slice of at least
// one io.ReadCloser's providing the stream(s) of bytes.
type Decompressor func([]byte, uint64, []io.ReadCloser) (io.ReadCloser, error)

var (
	//nolint:gochecknoglobals
	decompressors sync.Map

	errNeedOneReader = errors.New("copy: need exactly one reader")
)

func newCopyReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}
	// just return the passed io.ReadCloser)
	return readers[0], nil
}

//nolint:gochecknoinits
func init() {
	// Copy
	RegisterDecompressor([]byte{0x00}, Decompressor(newCopyReader))
	// Delta
	RegisterDecompressor([]byte{0x03}, Decompressor(delta.NewReader))
	// LZMA
	RegisterDecompressor([]byte{0x03, 0x01, 0x01}, Decompressor(lzma.NewReader))
	// BCJ
	RegisterDecompressor([]byte{0x03, 0x03, 0x01, 0x03}, Decompressor(bra.NewBCJReader))
	// BCJ2
	RegisterDecompressor([]byte{0x03, 0x03, 0x01, 0x1b}, Decompressor(bcj2.NewReader))
	// PPC
	RegisterDecompressor([]byte{0x03, 0x03, 0x02, 0x05}, Decompressor(bra.NewPPCReader))
	// ARM
	RegisterDecompressor([]byte{0x03, 0x03, 0x05, 0x01}, Decompressor(bra.NewARMReader))
	// SPARC
	RegisterDecompressor([]byte{0x03, 0x03, 0x08, 0x05}, Decompressor(bra.NewSPARCReader))
	// Deflate
	RegisterDecompressor([]byte{0x04, 0x01, 0x08}, Decompressor(deflate.NewReader))
	// Bzip2
	RegisterDecompressor([]byte{0x04, 0x02, 0x02}, Decompressor(bzip2.NewReader))
	// Zstandard
	RegisterDecompressor([]byte{0x04, 0xf7, 0x11, 0x01}, Decompressor(zstd.NewReader))
	// Brotli
	RegisterDecompressor([]byte{0x04, 0xf7, 0x11, 0x02}, Decompressor(brotli.NewReader))
	// LZ4
	RegisterDecompressor([]byte{0x04, 0xf7, 0x11, 0x04}, Decompressor(lz4.NewReader))
	// AES-CBC-256 & SHA-256
	RegisterDecompressor([]byte{0x06, 0xf1, 0x07, 0x01}, Decompressor(aes7z.NewReader))
	// LZMA2
	RegisterDecompressor([]byte{0x21}, Decompressor(lzma2.NewReader))
}

// RegisterDecompressor allows custom decompressors for a specified method ID.
func RegisterDecompressor(method []byte, dcomp Decompressor) {
	if _, dup := decompressors.LoadOrStore(string(method), dcomp); dup {
		panic("decompressor already registered")
	}
}

func decompressor(method []byte) Decompressor {
	di, ok := decompressors.Load(string(method))
	if !ok {
		return nil
	}

	if d, ok := di.(Decompressor); ok {
		return d
	}

	return nil
}
