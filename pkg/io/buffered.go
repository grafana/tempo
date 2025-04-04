package io

import (
	"errors"
	"io"
	"sync"
)

// BufferedReaderAt implements io.ReaderAt but extends and buffers reads up to the given buffer size.
// Subsequent reads are returned from the buffers. Additionally it supports concurrent readers
// by maintaining multiple buffers at different offsets, and matching up reads with existing
// buffers where possible. When needed the least-recently-used buffer is overwritten with new reads.
//
// Note that the locking effectively serializes reads to the underlying io.ReaderAt. This reader
// is not suitable for high-concurrency workloads, but does nicely reduce memory usage and backend calls
// for batch workloads.
type BufferedReaderAt struct {
	mtx     sync.Mutex
	ra      io.ReaderAt
	rasz    int64
	rdsz    int
	count   int64
	buffers []readerBuffer
}

type readerBuffer struct {
	buf   []byte
	off   int64
	count int64
}

var _ io.ReaderAt = (*BufferedReaderAt)(nil)

func NewBufferedReaderAt(ra io.ReaderAt, readerSize int64, bufSize, bufCount int) *BufferedReaderAt {
	r := &BufferedReaderAt{
		ra:      ra,
		rasz:    readerSize,
		rdsz:    bufSize,
		buffers: make([]readerBuffer, bufCount),
	}

	return r
}

func (r *BufferedReaderAt) canRead(buf *readerBuffer, offset, length int64) bool {
	return offset >= buf.off && (offset+length <= buf.off+int64(len(buf.buf)))
}

func (r *BufferedReaderAt) read(buf *readerBuffer, b []byte, offset int64) {
	start := offset - buf.off
	copy(b, buf.buf[start:start+int64(len(b))])
}

func (r *BufferedReaderAt) prep(buf *readerBuffer, offset, length int64) {
	offset, sz := calculateBounds(offset, length, r.rdsz, r.rasz)

	// Realloc?
	if int64(cap(buf.buf)) < sz {
		buf.buf = make([]byte, sz)
	}
	buf.buf = buf.buf[:sz]
	buf.off = offset
}

func (r *BufferedReaderAt) populate(buf *readerBuffer) (int, error) {
	// read
	n, err := r.ra.ReadAt(buf.buf, buf.off)

	// if err is fatal we need to invalidate the buffer by setting it back to 0s (uninitialized)
	if isFatalError(err) {
		buf.buf = buf.buf[:0]
		buf.count = 0
		buf.off = 0
	}

	return n, err
}

func calculateBounds(offset, length int64, bufferSize int, readerAtSize int64) (newOffset, newLength int64) {
	// Increase to minimim read size
	sz := length
	if sz < int64(bufferSize) {
		sz = int64(bufferSize)
	}

	// Don't read larger than entire contents
	if sz > readerAtSize {
		sz = readerAtSize
	}

	// If read extends past the end of reader,
	// back offset up to fill the whole buffer
	if offset+sz >= readerAtSize {
		offset = readerAtSize - sz
	}

	return offset, sz
}

func (r *BufferedReaderAt) ReadAt(b []byte, offset int64) (int, error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if len(r.buffers) == 0 {
		return r.ra.ReadAt(b, offset)
	}

	// Least-recently-used tracking
	r.count++
	var lru *readerBuffer
	var buf *readerBuffer
	for i := range r.buffers {
		if r.canRead(&r.buffers[i], offset, int64(len(b))) {
			buf = &r.buffers[i]
			break
		}

		if lru == nil || r.buffers[i].count < lru.count {
			lru = &r.buffers[i]
		}
	}

	if buf == nil {
		// No buffer satisfied read, overwrite least-recently-used
		buf = lru

		r.prep(buf, offset, int64(len(b)))
		buf.count = r.count

		var err error
		if _, err = r.populate(buf); isFatalError(err) {
			return 0, err
		}

		r.read(buf, b, offset)
		return len(b), err
	}

	buf.count = r.count
	r.read(buf, b, offset)
	return len(b), nil
}

type BufferedWriter struct {
	w   io.Writer
	buf []byte
}

type BufferedWriteFlusher interface {
	io.WriteCloser
	Len() int
	Flush() error
}

func NewBufferedWriter(w io.Writer) BufferedWriteFlusher {
	return &BufferedWriter{w, nil}
}

var _ BufferedWriteFlusher = (*BufferedWriter)(nil)

func (b *BufferedWriter) Write(p []byte) (n int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *BufferedWriter) Len() int {
	return len(b.buf)
}

func (b *BufferedWriter) Flush() error {
	n, err := b.w.Write(b.buf)
	b.buf = b.buf[:len(b.buf)-n]
	return err
}

func (b *BufferedWriter) Close() error {
	if len(b.buf) > 0 {
		err := b.Flush()
		b.buf = nil
		return err
	}
	return nil
}

func isFatalError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, io.EOF) {
		return false
	}

	return true
}
