// Package bzip2 implements the Bzip2 decompressor.
package bzip2

import (
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
)

type readCloser struct {
	c io.Closer
	r io.Reader
}

var (
	errAlreadyClosed = errors.New("bzip2: already closed")
	errNeedOneReader = errors.New("bzip2: need exactly one reader")
)

func (rc *readCloser) Close() error {
	if rc.c == nil || rc.r == nil {
		return errAlreadyClosed
	}

	if err := rc.c.Close(); err != nil {
		return fmt.Errorf("bzip2: error closing: %w", err)
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
		err = fmt.Errorf("bzip2: error reading: %w", err)
	}

	return n, err
}

// NewReader returns a new bzip2 io.ReadCloser.
func NewReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	return &readCloser{
		c: readers[0],
		r: bzip2.NewReader(readers[0]),
	}, nil
}
