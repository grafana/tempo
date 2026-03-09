package archives

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/klauspost/compress/s2"
)

func init() {
	RegisterFormat(Sz{})
}

// Sz facilitates Snappy compression. It uses S2
// for reading and writing, but by default will
// write Snappy-compatible data.
type Sz struct {
	// Configurable S2 extension.
	S2 S2
}

// S2 is an extension of Snappy that can read Snappy
// streams and write Snappy-compatible streams, but
// can also be configured to write Snappy-incompatible
// streams for greater gains. See
// https://pkg.go.dev/github.com/klauspost/compress/s2
// for details and the documentation for each option.
type S2 struct {
	// reader options
	MaxBlockSize           int
	AllocBlock             int
	IgnoreStreamIdentifier bool
	IgnoreCRC              bool

	// writer options
	AddIndex           bool
	Compression        S2Level
	BlockSize          int
	Concurrency        int
	FlushOnWrite       bool
	Padding            int
	SnappyIncompatible bool
}

func (Sz) Extension() string { return ".sz" }
func (Sz) MediaType() string { return "application/x-snappy-framed" }

func (sz Sz) Match(_ context.Context, filename string, stream io.Reader) (MatchResult, error) {
	var mr MatchResult

	// match filename
	if strings.Contains(strings.ToLower(filename), sz.Extension()) ||
		strings.Contains(strings.ToLower(filename), ".s2") {
		mr.ByName = true
	}

	// match file header
	buf, err := readAtMost(stream, len(snappyHeader))
	if err != nil {
		return mr, err
	}
	mr.ByStream = bytes.Equal(buf, snappyHeader)

	return mr, nil
}

func (sz Sz) OpenWriter(w io.Writer) (io.WriteCloser, error) {
	var opts []s2.WriterOption
	if sz.S2.AddIndex {
		opts = append(opts, s2.WriterAddIndex())
	}
	switch sz.S2.Compression {
	case S2LevelNone:
		opts = append(opts, s2.WriterUncompressed())
	case S2LevelBetter:
		opts = append(opts, s2.WriterBetterCompression())
	case S2LevelBest:
		opts = append(opts, s2.WriterBestCompression())
	}
	if sz.S2.BlockSize != 0 {
		opts = append(opts, s2.WriterBlockSize(sz.S2.BlockSize))
	}
	if sz.S2.Concurrency != 0 {
		opts = append(opts, s2.WriterConcurrency(sz.S2.Concurrency))
	}
	if sz.S2.FlushOnWrite {
		opts = append(opts, s2.WriterFlushOnWrite())
	}
	if sz.S2.Padding != 0 {
		opts = append(opts, s2.WriterPadding(sz.S2.Padding))
	}
	if !sz.S2.SnappyIncompatible {
		// this option is inverted because by default we should
		// probably write Snappy-compatible streams
		opts = append(opts, s2.WriterSnappyCompat())
	}
	return s2.NewWriter(w, opts...), nil
}

func (sz Sz) OpenReader(r io.Reader) (io.ReadCloser, error) {
	var opts []s2.ReaderOption
	if sz.S2.AllocBlock != 0 {
		opts = append(opts, s2.ReaderAllocBlock(sz.S2.AllocBlock))
	}
	if sz.S2.IgnoreCRC {
		opts = append(opts, s2.ReaderIgnoreCRC())
	}
	if sz.S2.IgnoreStreamIdentifier {
		opts = append(opts, s2.ReaderIgnoreStreamIdentifier())
	}
	if sz.S2.MaxBlockSize != 0 {
		opts = append(opts, s2.ReaderMaxBlockSize(sz.S2.MaxBlockSize))
	}
	return io.NopCloser(s2.NewReader(r, opts...)), nil
}

// Compression level for S2 (Snappy/Sz extension).
// EXPERIMENTAL: May be changed or removed without a major version bump.
type S2Level int

// Compression levels for S2.
// EXPERIMENTAL: May be changed or removed without a major version bump.
const (
	S2LevelNone   S2Level = 0
	S2LevelFast   S2Level = 1
	S2LevelBetter S2Level = 2
	S2LevelBest   S2Level = 3
)

// https://github.com/google/snappy/blob/master/framing_format.txt - contains "sNaPpY"
var snappyHeader = []byte{0xff, 0x06, 0x00, 0x00, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59}
