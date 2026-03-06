// Package deflate implements the Deflate decompressor.
package deflate

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/bodgit/sevenzip/internal/util"
	"github.com/hashicorp/go-multierror"
	"github.com/klauspost/compress/flate"
)

type readCloser struct {
	c  io.Closer
	fr io.ReadCloser
}

var (
	//nolint:gochecknoglobals
	flateReaderPool sync.Pool

	errAlreadyClosed = errors.New("deflate: already closed")
	errNeedOneReader = errors.New("deflate: need exactly one reader")
)

func (rc *readCloser) Close() error {
	if rc.c == nil || rc.fr == nil {
		return errAlreadyClosed
	}

	if err := multierror.Append(rc.fr.Close(), rc.c.Close()).ErrorOrNil(); err != nil {
		return fmt.Errorf("deflate: error closing: %w", err)
	}

	flateReaderPool.Put(rc.fr)
	rc.c, rc.fr = nil, nil

	return nil
}

func (rc *readCloser) Read(p []byte) (int, error) {
	if rc.c == nil || rc.fr == nil {
		return 0, errAlreadyClosed
	}

	n, err := rc.fr.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("deflate: error reading: %w", err)
	}

	return n, err
}

// NewReader returns a new DEFLATE io.ReadCloser.
func NewReader(_ []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	fr, ok := flateReaderPool.Get().(io.ReadCloser)
	if ok {
		frf, ok := fr.(flate.Resetter)
		if ok {
			if err := frf.Reset(util.ByteReadCloser(readers[0]), nil); err != nil {
				return nil, fmt.Errorf("deflate: error resetting: %w", err)
			}
		}
	} else {
		fr = flate.NewReader(util.ByteReadCloser(readers[0]))
	}

	return &readCloser{
		c:  readers[0],
		fr: fr,
	}, nil
}
