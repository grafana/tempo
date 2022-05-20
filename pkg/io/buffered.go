package io

import (
	"io"
	"sync"
)

// bufferedReader implements io.ReaderAt but extends and buffers reads up to the given buffer size.
// Subsequent reads are returned from the buffers. Additionally it supports concurrent readers
// by maintaining multiple buffers at different offsets, and matching up reads with existing
// buffers where possible. When needed the least-recently-used buffer is overwritten with new reads.
type bufferedReader struct {
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

var _ io.ReaderAt = (*bufferedReader)(nil)

func NewBufferedReaderAt(ra io.ReaderAt, readerSize int64, bufSize, bufCount int) *bufferedReader {
	r := &bufferedReader{
		ra:      ra,
		rasz:    readerSize,
		rdsz:    bufSize,
		buffers: make([]readerBuffer, bufCount),
	}

	return r
}

func (r *bufferedReader) canRead(buf *readerBuffer, offset, length int64) bool {
	return offset >= buf.off && (offset+length <= buf.off+int64(len(buf.buf)))
}

func (r *bufferedReader) read(buf *readerBuffer, b []byte, offset int64) {
	start := offset - buf.off
	copy(b, buf.buf[start:start+int64(len(b))])
}

func (r *bufferedReader) populate(buf *readerBuffer, offset, length int64) (int, error) {

	offset, sz := calculateBounds(offset, length, r.rdsz, r.rasz)

	// Realloc?
	if int64(cap(buf.buf)) < sz {
		buf.buf = make([]byte, sz)
	}
	buf.buf = buf.buf[:sz]

	// Read
	buf.off = offset
	n, err := r.ra.ReadAt(buf.buf, offset)
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

func (r *bufferedReader) ReadAt(b []byte, offset int64) (int, error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if len(r.buffers) == 0 {
		return r.ra.ReadAt(b, offset)
	}

	// Least-recently-used tracking
	r.count++
	var lru *readerBuffer

	for i := range r.buffers {
		buf := &r.buffers[i]
		if r.canRead(buf, offset, int64(len(b))) {
			r.read(buf, b, offset)
			r.buffers[i].count = r.count
			return len(b), nil
		}

		if lru == nil || r.buffers[i].count < lru.count {
			lru = buf
		}
	}

	// Need to read, overwrite least-recently-used
	buf := lru
	if _, err := r.populate(buf, offset, int64(len(b))); err != nil {
		return 0, err
	}

	buf.count = r.count
	r.read(buf, b, offset)
	return len(b), nil
}

type BufferedWriter struct {
	w   io.Writer
	buf []byte
}

func NewBufferedWriter(w io.Writer) *BufferedWriter {
	return &BufferedWriter{w, nil}
}

var _ io.WriteCloser = (*BufferedWriter)(nil)

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
