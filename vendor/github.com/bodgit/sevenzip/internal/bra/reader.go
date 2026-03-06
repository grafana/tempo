package bra

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type readCloser struct {
	rc   io.ReadCloser
	buf  bytes.Buffer
	n    int
	conv converter
}

var (
	errAlreadyClosed = errors.New("bra: already closed")
	errNeedOneReader = errors.New("bra: need exactly one reader")
)

func (rc *readCloser) Close() error {
	if rc.rc == nil {
		return errAlreadyClosed
	}

	if err := rc.rc.Close(); err != nil {
		return fmt.Errorf("bra: error closing: %w", err)
	}

	rc.rc = nil

	return nil
}

func (rc *readCloser) Read(p []byte) (int, error) {
	if rc.rc == nil {
		return 0, errAlreadyClosed
	}

	if _, err := io.CopyN(&rc.buf, rc.rc, int64(max(len(p), rc.conv.Size())-rc.buf.Len())); err != nil {
		if !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("bra: error buffering: %w", err)
		}

		if rc.buf.Len() < rc.conv.Size() {
			rc.n = rc.buf.Len()
		}
	}

	rc.n += rc.conv.Convert(rc.buf.Bytes()[rc.n:], false)

	n, err := rc.buf.Read(p[:min(rc.n, len(p))])
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("bra: error reading: %w", err)
	}

	rc.n -= n

	return n, err
}

func newReader(readers []io.ReadCloser, conv converter) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	return &readCloser{
		rc:   readers[0],
		conv: conv,
	}, nil
}
