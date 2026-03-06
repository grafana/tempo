// Package zstd implements the Zstandard decompressor.
package zstd

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type readCloser struct {
	c io.Closer
	r *zstd.Decoder
}

var (
	//nolint:gochecknoglobals
	zstdReaderPool sync.Pool

	errAlreadyClosed = errors.New("zstd: already closed")
	errNeedOneReader = errors.New("zstd: need exactly one reader")
)

func (rc *readCloser) Close() error {
	if rc.c == nil {
		return errAlreadyClosed
	}

	if err := rc.c.Close(); err != nil {
		return fmt.Errorf("zstd: error closing: %w", err)
	}

	zstdReaderPool.Put(rc.r)
	rc.c, rc.r = nil, nil

	return nil
}

func (rc *readCloser) Read(p []byte) (int, error) {
	if rc.r == nil {
		return 0, errAlreadyClosed
	}

	n, err := rc.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("zstd: error reading: %w", err)
	}

	return n, err
}

// NewReader returns a new Zstandard io.ReadCloser.
func NewReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	var err error

	r, ok := zstdReaderPool.Get().(*zstd.Decoder)
	if ok {
		if err = r.Reset(readers[0]); err != nil {
			return nil, fmt.Errorf("zstd: error resetting: %w", err)
		}
	} else {
		if r, err = zstd.NewReader(readers[0]); err != nil {
			return nil, fmt.Errorf("zstd: error creating reader: %w", err)
		}

		runtime.SetFinalizer(r, (*zstd.Decoder).Close)
	}

	return &readCloser{
		c: readers[0],
		r: r,
	}, nil
}
