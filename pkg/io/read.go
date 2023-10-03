package io

import (
	"errors"
	"io"
)

// ReadAllWithEstimate is a fork of https://go.googlesource.com/go/+/go1.16.3/src/io/io.go#626
// with a starting buffer size. if none is provided it uses the existing default of 512
func ReadAllWithEstimate(r io.Reader, estimatedBytes int64) ([]byte, error) {
	if estimatedBytes <= 0 {
		estimatedBytes = 512
	}

	b := make([]byte, 0, estimatedBytes+1) // if the calling code knows the exact bytes needed the below logic will do one extra allocation unless we add 1
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return b, err
		}
	}
}

// ReadAllWithBuffer is a fork of https://go.googlesource.com/go/+/go1.16.3/src/io/io.go#626
// with a buffer to read into. if the provided buffer is not large enough it will be extended
// and returned to the caller
func ReadAllWithBuffer(r io.Reader, estimatedBytes int, b []byte) ([]byte, error) {
	if estimatedBytes == 0 {
		estimatedBytes = 512
	}

	if cap(b) < estimatedBytes {
		b = make([]byte, 0, estimatedBytes+1) // if the calling code knows the exact bytes needed the below logic will do one extra allocation unless we add 1
	} else {
		b = b[0:0]
	}

	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return b, err
		}
	}
}
