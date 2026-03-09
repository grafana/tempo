// Package delta implements the Delta filter.
package delta

import (
	"errors"
	"fmt"
	"io"
)

type readCloser struct {
	rc    io.ReadCloser
	state [stateSize]byte
	delta int
}

const (
	stateSize = 256
)

var (
	errAlreadyClosed          = errors.New("delta: already closed")
	errNeedOneReader          = errors.New("delta: need exactly one reader")
	errInsufficientProperties = errors.New("delta: not enough properties")
)

func (rc *readCloser) Close() error {
	if rc.rc == nil {
		return errAlreadyClosed
	}

	if err := rc.rc.Close(); err != nil {
		return fmt.Errorf("delta: error closing: %w", err)
	}

	rc.rc = nil

	return nil
}

func (rc *readCloser) Read(p []byte) (int, error) {
	if rc.rc == nil {
		return 0, errAlreadyClosed
	}

	n, err := rc.rc.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("delta: error reading: %w", err)
	}

	var (
		buffer [stateSize]byte
		j      int
	)

	copy(buffer[:], rc.state[:rc.delta])

	for i := 0; i < n; {
		for j = 0; j < rc.delta && i < n; i++ {
			p[i] = buffer[j] + p[i]
			buffer[j] = p[i]
			j++
		}
	}

	if j == rc.delta {
		j = 0
	}

	copy(rc.state[:], buffer[j:rc.delta])
	copy(rc.state[rc.delta-j:], buffer[:j])

	return n, nil
}

// NewReader returns a new Delta io.ReadCloser.
func NewReader(p []byte, _ uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(readers) != 1 {
		return nil, errNeedOneReader
	}

	if len(p) != 1 {
		return nil, errInsufficientProperties
	}

	return &readCloser{
		rc:    readers[0],
		delta: int(p[0] + 1),
	}, nil
}
