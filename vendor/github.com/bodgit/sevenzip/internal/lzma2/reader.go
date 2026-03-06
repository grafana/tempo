// Package lzma2 implements the LZMA2 decompressor.
package lzma2

import (
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
	errAlreadyClosed          = errors.New("lzma2: already closed")
	errNeedOneReader          = errors.New("lzma2: need exactly one reader")
	errInsufficientProperties = errors.New("lzma2: not enough properties")
)

func (rc *readCloser) Close() error {
	if rc.c == nil || rc.r == nil {
		return errAlreadyClosed
	}

	if err := rc.c.Close(); err != nil {
		return fmt.Errorf("lzma2: error closing: %w", err)
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
		err = fmt.Errorf("lzma2: error reading: %w", err)
	}

	return n, err
}

// NewReader returns a new LZMA2 io.ReadCloser.
func NewReader(p []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	if len(p) != 1 {
		return nil, errInsufficientProperties
	}

	config := lzma.Reader2Config{
		DictCap: (2 | (int(p[0]) & 1)) << (p[0]/2 + 11), // This gem came from Lzma2Dec.c
	}

	if err := config.Verify(); err != nil {
		return nil, fmt.Errorf("lzma2: error verifying config: %w", err)
	}

	lr, err := config.NewReader2(readers[0])
	if err != nil {
		return nil, fmt.Errorf("lzma2: error creating reader: %w", err)
	}

	return &readCloser{
		c: readers[0],
		r: lr,
	}, nil
}
