package io

import (
	"io"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

// BufferedReaderAt implements io.ReaderAt but extends and buffers reads up to the given buffer size.
// Subsequent reads are returned from the buffers. Additionally it supports concurrent readers
// by maintaining multiple buffers at different offsets, and matching up reads with existing
// buffers where possible. When needed the least-recently-used buffer is overwritten with new reads.
type BufferedReaderAt struct {
	mtx     sync.Mutex
	ra      io.ReaderAt
	rasz    int64
	rdsz    int
	count   int64
	buffers []readerBuffer
}

type readerBuffer struct {
	mtx   sync.RWMutex
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
	// Read
	n, err := r.ra.ReadAt(buf.buf, buf.off)
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
		//offset = readerAtSize - sz
		// jpe - reduces head of line blocking by only pulling exactly the footer when requested
		sz = readerAtSize - offset
	}

	return offset, sz
}

func (r *BufferedReaderAt) ReadAt(b []byte, offset int64) (int, error) {
	// There are two-levels of locking: the top-level governs the
	// the reader and the arrangement and position of the buffers.
	// Then each individual buffer has its own lock for populating
	// and reading it.

	// The main reason for this is to support concurrent activity
	// while solving the stampeding herd issue for fresh reads:
	// The first read will prep the offset/length of the buffer
	// and then switch to the buffer's write-lock while populating it.
	// The second read will inspect the offset/length and know
	// that it will satisfy, but by taking the read-lock will
	// wait until the first call has finished populating the buffer .

	s := recordStart(len(b), offset)
	defer func() {
		recordDone(s)
		dumpStats(s)
	}()
	r.mtx.Lock()
	recordReaderLockAcquired(s)

	if len(r.buffers) == 0 {
		r.mtx.Unlock()
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

	recordBufFound(s, buf)

	if buf == nil {
		// No buffer satisfied read, overwrite least-recently-used
		buf = lru

		// Here we exchange the top-level lock for
		// the buffer's individual write lock
		buf.mtx.Lock()
		defer buf.mtx.Unlock()
		r.prep(buf, offset, int64(len(b)))
		buf.count = r.count
		r.mtx.Unlock()

		recordBufLock(s)

		if _, err := r.populate(buf); err != nil {
			return 0, err
		}

		r.read(buf, b, offset)
		return len(b), nil
	}

	// Here we exchange the top-level lock for
	// the buffer's individual read lock
	buf.mtx.RLock()
	defer buf.mtx.RUnlock()
	buf.count = r.count
	r.mtx.Unlock()

	recordBufLock(s)

	r.read(buf, b, offset)
	return len(b), nil
}

type BufferedWriter struct {
	w   io.Writer
	buf []byte
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

type BufferedWriteFlusher interface {
	io.WriteCloser
	Len() int
	Flush() error
}

// BufferedWriterWithQueue is an attempt at writing an async queue of outgoing data to the underlying writer.
// As a tradeoff for removing flushes from the hot path, this writer does not provide guarantees about
// bubbling up errors and takes a best effort approach to signal a failure.
//
// Note: This is not used at the moment due to concerns with error handling when writing async data to the backend.
type BufferedWriterWithQueue struct {
	w   io.Writer
	buf []byte

	flushCh chan []byte
	doneCh  chan struct{}
	err     atomic.Error
}

var _ BufferedWriteFlusher = (*BufferedWriterWithQueue)(nil)

func NewBufferedWriterWithQueue(w io.Writer) BufferedWriteFlusher {
	b := &BufferedWriterWithQueue{
		w:       w,
		buf:     nil,
		flushCh: make(chan []byte, 10), // todo: guess better?
		doneCh:  make(chan struct{}, 1),
	}

	go b.flushLoop()

	return b
}

func (b *BufferedWriterWithQueue) Write(p []byte) (n int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *BufferedWriterWithQueue) Len() int {
	return len(b.buf)
}

func (b *BufferedWriterWithQueue) Flush() error {
	if err := b.err.Load(); err != nil {
		return errors.Wrap(err, "error in async write using buffered writer")
	}

	bufCopy := make([]byte, 0, len(b.buf))
	bufCopy = append(bufCopy, b.buf...)

	// reset/resize buffer
	b.buf = b.buf[:0]

	// will only block if the entire buffered channel is full
	b.flushCh <- bufCopy
	return nil
}

func (b *BufferedWriterWithQueue) flushLoop() {
	defer close(b.doneCh)

	// for-range will exit once channel is closed
	// https://dave.cheney.net/tag/golang-3
	for buf := range b.flushCh {
		_, err := b.w.Write(buf)
		if err != nil {
			b.err.Store(err)
			return
		}
	}
}

func (b *BufferedWriterWithQueue) Close() error {
	if err := b.err.Load(); err != nil {
		return err
	}

	var flushErr error
	if len(b.buf) > 0 {
		flushErr = b.Flush()
		b.buf = nil
	}

	// close out flushLoop
	close(b.flushCh)

	// blocking wait on doneCh
	<-b.doneCh

	return flushErr
}
