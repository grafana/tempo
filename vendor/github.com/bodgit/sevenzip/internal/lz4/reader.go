// Package lz4 implements the LZ4 decompressor.
package lz4

import (
	"errors"
	"fmt"
	"io"
	"sync"

	lz4 "github.com/pierrec/lz4/v4"
)

type readCloser struct {
	c io.Closer
	r *lz4.Reader
}

var (
	//nolint:gochecknoglobals
	lz4ReaderPool sync.Pool

	errAlreadyClosed = errors.New("lz4: already closed")
	errNeedOneReader = errors.New("lz4: need exactly one reader")
)

func (rc *readCloser) Close() error {
	if rc.c == nil || rc.r == nil {
		return errAlreadyClosed
	}

	if err := rc.c.Close(); err != nil {
		return fmt.Errorf("lz4: error closing: %w", err)
	}

	lz4ReaderPool.Put(rc.r)
	rc.c, rc.r = nil, nil

	return nil
}

func (rc *readCloser) Read(p []byte) (int, error) {
	if rc.r == nil {
		return 0, errAlreadyClosed
	}

	n, err := rc.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("lz4: error reading: %w", err)
	}

	return n, err
}

// NewReader returns a new LZ4 io.ReadCloser.
func NewReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	r, ok := lz4ReaderPool.Get().(*lz4.Reader)
	if ok {
		r.Reset(readers[0])
	} else {
		r = lz4.NewReader(readers[0])
	}

	return &readCloser{
		c: readers[0],
		r: r,
	}, nil
}
