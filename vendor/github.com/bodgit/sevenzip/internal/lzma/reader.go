// Package lzma implements the LZMA decompressor.
package lzma

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ulikunitz/xz/lzma"
)

type readCloser struct {
	c io.Closer
	r io.Reader
}

var (
	errAlreadyClosed = errors.New("lzma: already closed")
	errNeedOneReader = errors.New("lzma: need exactly one reader")
)

func (rc *readCloser) Close() error {
	if rc.c == nil || rc.r == nil {
		return errAlreadyClosed
	}

	if err := rc.c.Close(); err != nil {
		return fmt.Errorf("lzma: error closing: %w", err)
	}

	rc.c, rc.r = nil, nil

	return nil
}

func (rc *readCloser) Read(p []byte) (int, error) {
	if rc.r == nil {
		return 0, errAlreadyClosed
	}

	n, err := rc.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("lzma: error reading: %w", err)
	}

	return n, err
}

// NewReader returns a new LZMA io.ReadCloser.
func NewReader(p []byte, s uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	h := bytes.NewBuffer(p)
	_ = binary.Write(h, binary.LittleEndian, s)

	lr, err := lzma.NewReader(multiReader(h, readers[0]))
	if err != nil {
		return nil, fmt.Errorf("lzma: error creating reader: %w", err)
	}

	return &readCloser{
		c: readers[0],
		r: lr,
	}, nil
}

func multiReader(b *bytes.Buffer, rc io.ReadCloser) io.Reader {
	mr := io.MultiReader(b, rc)

	if br, ok := rc.(io.ByteReader); ok {
		return &multiByteReader{
			b:  b,
			br: br,
			mr: mr,
		}
	}

	return mr
}

type multiByteReader struct {
	b  *bytes.Buffer
	br io.ByteReader
	mr io.Reader
}

func (m *multiByteReader) ReadByte() (b byte, err error) {
	if m.b.Len() > 0 {
		b, err = m.b.ReadByte()
	} else {
		b, err = m.br.ReadByte()
	}

	if err != nil {
		err = fmt.Errorf("lzma: error multi byte reading: %w", err)
	}

	return b, err
}

func (m *multiByteReader) Read(p []byte) (int, error) {
	n, err := m.mr.Read(p)
	if err != nil {
		err = fmt.Errorf("lzma: error multi reading: %w", err)
	}

	return n, err
}
