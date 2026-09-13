package lh

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/olekukonko/ll/lx"
)

// trackingWriter wraps an io.WriteCloser to keep an in-memory count of bytes written.
// This prevents the rotator from having to query the filesystem (via os.Stat)
// on every single log entry, which would cause severe performance bottlenecks.
type trackingWriter struct {
	io.WriteCloser
	written int64 // Atomic: use atomic.LoadInt64/AddInt64
}

// Write intercepts the write operation, counts the bytes, and passes them to the underlying writer.
func (t *trackingWriter) Write(p []byte) (n int, err error) {
	n, err = t.WriteCloser.Write(p)
	if n > 0 {
		atomic.AddInt64(&t.written, int64(n))
	}
	return
}

// writtenBytes returns the current byte count atomically.
func (t *trackingWriter) writtenBytes() int64 {
	if t == nil {
		return 0
	}
	return atomic.LoadInt64(&t.written)
}

// RotateSource defines the callbacks needed to implement log rotation.
// It abstracts the destination lifecycle: opening, sizing, and rotating.
//
// Example for file rotation:
//
//	src := lh.RotateSource{
//		Open: func() (io.WriteCloser, error) {
//			return os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
//		},
//		Size: func() (int64, error) {
//			if fi, err := os.Stat("app.log"); err == nil {
//				return fi.Size(), nil
//			}
//			return 0, nil // File doesn't exist yet
//		},
//		Rotate: func() error {
//			// Close and rename the current log before creating a new one.
//			return os.Rename("app.log", "app.log."+time.Now().Format("20060102-150405"))
//		},
//	}
type RotateSource struct {
	// Open returns a fresh destination for log output.
	// Called on initialization and after each rotation.
	Open func() (io.WriteCloser, error)

	// Size returns the current size in bytes of the active destination.
	// Return an error if size cannot be determined (rotation will be skipped).
	Size func() (int64, error)

	// Rotate performs all cleanup/rotation actions before a new destination is
	// opened, including closing or renaming the previous writer when required.
	// Rotating will NOT close the old writer itself; that is the responsibility
	// of this callback.  May be nil if no pre-open actions are needed.
	Rotate func() error
}

// Rotating wraps a handler to rotate its output when maxSize is exceeded.
// The wrapped handler must implement both Handler and Outputter interfaces.
// Rotation is triggered on each Handle call if the current size >= maxSize.
//
// Example:
//
//	handler := lx.NewJSONHandler(os.Stdout)
//	src := lh.RotateSource{...} // see RotateSource example
//	rotator, err := lh.NewRotating(handler, 10*1024*1024, src) // 10 MB
//	logger := lx.NewLogger(rotator)
//	logger.Info("This log may trigger rotation when file reaches 10MB")
type Rotating[H interface {
	lx.Handler
	lx.Outputter
}] struct {
	mu      sync.Mutex
	maxSize int64
	src     RotateSource

	out     *trackingWriter // Uses the tracking wrapper to count bytes in memory
	handler H
}

// NewRotating creates a rotating wrapper around handler.
// Handler's output will be replaced with destinations from src.Open.
// If maxSizeBytes <= 0, rotation is disabled.
// src.Rotate may be nil if no pre-open actions are needed.
//
// Example:
//
//	// Create a JSON handler that rotates at 5MB
//	handler := lx.NewJSONHandler(os.Stdout)
//	rotator, err := lh.NewRotating(handler, 5*1024*1024, src)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Use rotator as your logger's handler
//	logger := lx.NewLogger(rotator)
func NewRotating[H interface {
	lx.Handler
	lx.Outputter
}](handler H, maxSizeBytes int64, src RotateSource) (*Rotating[H], error) {
	// Validate that Open callback is provided
	if src.Open == nil {
		return nil, io.ErrClosedPipe
	}

	r := &Rotating[H]{
		maxSize: maxSizeBytes,
		src:     src,
		handler: handler,
	}
	if err := r.reopenLocked(); err != nil {
		return nil, err
	}
	return r, nil
}

// Handle processes a log entry, rotating output if necessary.
// Thread-safe: can be called concurrently.
//
// Example:
//
//	rotator.Handle(&lx.Entry{
//	    Level:     lx.InfoLevel,
//	    Message:   "Processing request",
//	    Namespace: "api",
//	})
func (r *Rotating[H]) Handle(e *lx.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.rotateIfNeededLocked(); err != nil {
		return err
	}
	return r.handler.Handle(e)
}

// Close releases resources (closes the current output).
// Safe to call multiple times.
//
// Example:
//
//	defer rotator.Close()
func (r *Rotating[H]) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.out != nil {
		return r.out.Close()
	}
	return nil
}

// Written returns the total bytes written to the current output destination.
// Useful for metrics and monitoring.
func (r *Rotating[H]) Written() int64 {
	r.mu.Lock()
	out := r.out
	r.mu.Unlock()
	return out.writtenBytes()
}

// rotateIfNeededLocked checks current size and rotates if maxSize exceeded.
// Called with mu already held.
//
// The old trackingWriter is simply dereferenced (not closed) because ownership
// of the underlying io.WriteCloser belongs to the src.Rotate callback.  That
// callback is responsible for closing, renaming, or otherwise finishing with
// the old destination before src.Open is called to provide a fresh one.  This
// design avoids double-closes on shared writers (e.g. test mocks, pipes) and
// correctly models real file-rotation where the OS rename is done before the
// old fd is released.
func (r *Rotating[H]) rotateIfNeededLocked() error {
	if r.maxSize <= 0 || r.src.Open == nil {
		return nil
	}

	// PERFORMANCE OPTIMIZATION:
	// Instead of calling r.src.Size() (which executes a slow os.Stat filesystem call),
	// we simply check our fast, in-memory integer counter.
	if r.out != nil && r.out.writtenBytes() < r.maxSize {
		return nil
	}

	// Drop the reference to the old trackingWriter without closing the underlying
	// WriteCloser.  Closing/renaming is the responsibility of src.Rotate (see doc above).
	r.out = nil

	// Run rotation hook (rename/move/compress/close old file, etc.)
	if r.src.Rotate != nil {
		if err := r.src.Rotate(); err != nil {
			return err
		}
	}

	// Open fresh output
	return r.reopenLocked()
}

// reopenLocked opens a new destination and sets it on the handler.
// Called with mu already held.
func (r *Rotating[H]) reopenLocked() error {
	out, err := r.src.Open()
	if err != nil {
		return err
	}

	// We only ask the filesystem for the true file size ONCE when we first open the file.
	// This is necessary to know the starting size if we are appending to an existing log file.
	var initialSize int64
	if r.src.Size != nil {
		initialSize, _ = r.src.Size()
	}

	// Wrap the returned io.WriteCloser so we can track all future bytes written in memory.
	r.out = &trackingWriter{
		WriteCloser: out,
		written:     initialSize,
	}

	r.handler.Output(r.out)
	return nil
}
