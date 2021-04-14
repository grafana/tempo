package io

import "io"

// ReadAllWithEstimate is a fork of https://go.googlesource.com/go/+/go1.16.3/src/io/io.go#626
//  with a starting buffer size. if none is provided it uses the existing default of 512
func ReadAllWithEstimate(r io.Reader, estimatedBytes int) ([]byte, error) {
	if estimatedBytes == 0 {
		estimatedBytes = 512
	}

	b := make([]byte, 0, estimatedBytes)
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}
