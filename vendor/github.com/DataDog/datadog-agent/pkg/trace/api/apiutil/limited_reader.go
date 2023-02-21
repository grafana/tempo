// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package apiutil

import (
	"errors"
	"io"
)

// ErrLimitedReaderLimitReached indicates that the read limit has been
// reached.
var ErrLimitedReaderLimitReached = errors.New("read limit reached")

// LimitedReader reads from a reader up to a specific limit. When this limit
// has been reached, any subsequent read will return
// ErrLimitedReaderLimitReached.
// The underlying reader has to implement io.ReadCloser so that it can be used
// with http request bodies.
type LimitedReader struct {
	r     io.ReadCloser
	limit int64
	Count int64
}

// NewLimitedReader creates a new LimitedReader.
func NewLimitedReader(r io.ReadCloser, limit int64) *LimitedReader {
	return &LimitedReader{
		r:     r,
		limit: limit,
	}
}

// Read reads from the underlying reader.
func (r *LimitedReader) Read(buf []byte) (n int, err error) {
	if r.limit <= 0 {
		return 0, ErrLimitedReaderLimitReached
	}

	if int64(len(buf)) > r.limit {
		buf = buf[0:r.limit]
	}
	n, err = r.r.Read(buf)

	// Some libraries (e.g. msgp) will ignore read data if err is not nil.
	// We reset err if something was read, and the next read will return
	// io.EOF with no data.
	if err == io.EOF && n > 0 {
		err = nil
	}

	r.limit -= int64(n)
	r.Count += int64(n)
	return
}

// Close closes the underlying reader.
func (r *LimitedReader) Close() error {
	return r.r.Close()
}
