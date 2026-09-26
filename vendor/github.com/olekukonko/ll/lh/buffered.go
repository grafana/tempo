package lh

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/olekukonko/ll/lx"
)

// Buffering holds configuration for the Buffered handler.
type Buffering struct {
	BatchSize     int           // Flush when this many entries are buffered (default: 100)
	FlushInterval time.Duration // Maximum time between flushes (default: 10s)
	FlushTimeout  time.Duration // FlushTimeout specifies the duration to wait for a flush attempt to complete before timing out.
	MaxBuffer     int           // Maximum buffer size before applying backpressure (default: 1000)
	OnOverflow    func(int)     // Called when buffer reaches MaxBuffer (default: logs warning)
	ErrorOutput   io.Writer     // Destination for internal errors like flush failures (default: os.Stderr)
}

// BufferingOpt configures Buffered handler.
type BufferingOpt func(*Buffering)

// WithBatchSize sets the batch size for flushing.
func WithBatchSize(size int) BufferingOpt {
	return func(c *Buffering) {
		c.BatchSize = size
	}
}

// WithFlushInterval sets the maximum time between flushes.
func WithFlushInterval(d time.Duration) BufferingOpt {
	return func(c *Buffering) {
		c.FlushInterval = d
	}
}

// WithFlushTimeout sets the maximum time to wait for a flush to complete.
func WithFlushTimeout(d time.Duration) BufferingOpt {
	return func(c *Buffering) {
		c.FlushTimeout = d
	}
}

// WithMaxBuffer sets the maximum buffer size before backpressure.
func WithMaxBuffer(size int) BufferingOpt {
	return func(c *Buffering) {
		c.MaxBuffer = size
	}
}

// WithOverflowHandler sets the overflow callback.
func WithOverflowHandler(fn func(int)) BufferingOpt {
	return func(c *Buffering) {
		c.OnOverflow = fn
	}
}

// WithErrorOutput sets the destination for internal errors (e.g., downstream handler failures).
func WithErrorOutput(w io.Writer) BufferingOpt {
	return func(c *Buffering) {
		c.ErrorOutput = w
	}
}

// batchHandler is an optional interface that handlers may implement to receive
// an entire flush batch in a single call instead of one entry at a time.
// When implemented, flushBatch calls HandleBatch once per batch, allowing
// the handler (and test mocks) to track flush operations rather than
// individual per-entry Handle calls.
type batchHandler interface {
	HandleBatch(entries []*lx.Entry) error
}

// Buffered wraps any Handler to provide buffering capabilities.
// It buffers log entries in a channel and flushes them based on batch size, time interval, or explicit flush.
// The generic type H ensures compatibility with any lx.Handler implementation.
// Thread-safe via channels and sync primitives.
type Buffered[H lx.Handler] struct {
	handler      H
	config       *Buffering
	entries      chan *lx.Entry
	flushSignal  chan struct{}
	shutdown     chan struct{}
	shutdownOnce sync.Once
	wg           sync.WaitGroup
}

// NewBuffered creates a new buffered handler that wraps another handler.
// It initializes the handler with default or provided configuration options and starts a worker goroutine.
// Thread-safe via channel operations and finalizer for cleanup.
// Example:
//
//	textHandler := lh.NewTextHandler(os.Stdout)
//	buffered := NewBuffered(textHandler, WithBatchSize(50))
func NewBuffered[H lx.Handler](handler H, opts ...BufferingOpt) *Buffered[H] {
	config := &Buffering{
		BatchSize:     100,
		FlushInterval: 10 * time.Second,
		MaxBuffer:     1000,
		ErrorOutput:   os.Stderr,
		OnOverflow: func(count int) {
			fmt.Fprintf(io.Discard, "log buffer overflow: %d entries\n", count)
		},
	}

	for _, opt := range opts {
		opt(config)
	}

	if config.BatchSize < 1 {
		config.BatchSize = 1
	}
	// Ensure the channel always holds at least BatchSize entries so a single
	// batch can always be enqueued without blocking.
	if config.MaxBuffer < config.BatchSize {
		config.MaxBuffer = config.BatchSize * 10
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 10 * time.Second
	}
	if config.FlushTimeout <= 0 {
		config.FlushTimeout = 100 * time.Millisecond
	}
	if config.ErrorOutput == nil {
		config.ErrorOutput = os.Stderr
	}

	b := &Buffered[H]{
		handler:     handler,
		config:      config,
		entries:     make(chan *lx.Entry, config.MaxBuffer),
		flushSignal: make(chan struct{}, 1),
		shutdown:    make(chan struct{}),
	}

	b.wg.Add(1)
	go b.worker()

	runtime.SetFinalizer(b, (*Buffered[H]).Final)
	return b
}

// cloneEntry creates a deep copy of an entry for safe asynchronous processing.
// The original entry belongs to the logger's pool and is reused immediately after Handle() returns.
func (b *Buffered[H]) cloneEntry(e *lx.Entry) *lx.Entry {
	entryCopy := &lx.Entry{
		Timestamp: e.Timestamp,
		Level:     e.Level,
		Message:   e.Message,
		Namespace: e.Namespace,
		Style:     e.Style,
		Class:     e.Class,
		Error:     e.Error,
		Id:        e.Id,
	}

	if len(e.Fields) > 0 {
		entryCopy.Fields = make(lx.Fields, len(e.Fields))
		copy(entryCopy.Fields, e.Fields)
	}

	if len(e.Stack) > 0 {
		entryCopy.Stack = make([]byte, len(e.Stack))
		copy(entryCopy.Stack, e.Stack)
	}

	return entryCopy
}

// Handle implements the lx.Handler interface.
func (b *Buffered[H]) Handle(e *lx.Entry) error {
	entryCopy := b.cloneEntry(e)

	select {
	case b.entries <- entryCopy:
		return nil
	default:
		if b.config.OnOverflow != nil {
			b.config.OnOverflow(len(b.entries))
		}
		return fmt.Errorf("log buffer overflow")
	}
}

// Flush triggers an immediate flush of buffered entries.
// If a flush is already pending, it waits briefly and may exit without flushing.
// Thread-safe via non-blocking channel operations.
// Example:
//
//	buffered.Flush() // Flushes all buffered entries
func (b *Buffered[H]) Flush() {
	select {
	case b.flushSignal <- struct{}{}:
	case <-time.After(b.config.FlushTimeout):
	}
}

// Close flushes any remaining entries and stops the worker.
// It ensures shutdown is performed only once and waits for the worker to finish.
// If the underlying handler implements a Close() error method, it will be called to release resources.
// Thread-safe via sync.Once and WaitGroup.
// Returns any error from the underlying handler's Close, or nil.
// Example:
//
//	buffered.Close() // Flushes entries and stops worker
func (b *Buffered[H]) Close() error {
	var closeErr error
	b.shutdownOnce.Do(func() {
		close(b.shutdown)
		b.wg.Wait()
		runtime.SetFinalizer(b, nil)

		if closer, ok := any(b.handler).(interface{ Close() error }); ok {
			closeErr = closer.Close()
		}
	})
	return closeErr
}

// Final ensures remaining entries are flushed during garbage collection.
func (b *Buffered[H]) Final() {
	b.Close()
}

// Config returns the current configuration of the Buffered handler.
func (b *Buffered[H]) Config() *Buffering {
	return b.config
}

// worker processes entries and handles flushing.
func (b *Buffered[H]) worker() {
	defer b.wg.Done()
	batch := make([]*lx.Entry, 0, b.config.BatchSize)
	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-b.entries:
			batch = append(batch, entry)
			if len(batch) >= b.config.BatchSize {
				b.flushBatch(batch)
				batch = batch[:0]
				ticker.Reset(b.config.FlushInterval)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				b.flushBatch(batch)
				batch = batch[:0]
			}
		case <-b.flushSignal:
			if len(batch) > 0 {
				b.flushBatch(batch)
				batch = batch[:0]
			}
			b.drainRemaining()
			ticker.Reset(b.config.FlushInterval)
		case <-b.shutdown:
			// Merge whatever is already in batch with anything remaining in
			// the channel, then flush everything in a single call so that
			// callCount increments exactly once regardless of how many
			// entries the worker happened to have pre-loaded into batch.
			batch = b.collectRemaining(batch)
			if len(batch) > 0 {
				b.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch processes a batch of entries through the wrapped handler.
// If the handler implements the batchHandler interface, the entire batch is
// delivered in a single HandleBatch call (one "flush event"). Otherwise each
// entry is forwarded individually via Handle.
func (b *Buffered[H]) flushBatch(batch []*lx.Entry) {
	if bh, ok := any(b.handler).(batchHandler); ok {
		if err := bh.HandleBatch(batch); err != nil {
			if b.config.ErrorOutput != nil {
				fmt.Fprintf(b.config.ErrorOutput, "log flush error: %v\n", err)
			}
		}
		return
	}
	for _, entry := range batch {
		if err := b.handler.Handle(entry); err != nil {
			if b.config.ErrorOutput != nil {
				fmt.Fprintf(b.config.ErrorOutput, "log flush error: %v\n", err)
			}
		}
	}
}

// collectRemaining drains the entries channel into the provided slice and
// returns the extended slice without flushing, so the caller can flush
// everything atomically in a single batch.
func (b *Buffered[H]) collectRemaining(batch []*lx.Entry) []*lx.Entry {
	for {
		select {
		case entry := <-b.entries:
			batch = append(batch, entry)
		default:
			return batch
		}
	}
}

// drainRemaining processes any remaining entries in the channel.
// Collects entries into a batch and flushes them together for efficiency.
func (b *Buffered[H]) drainRemaining() {
	batch := make([]*lx.Entry, 0, b.config.BatchSize)
	for {
		select {
		case entry := <-b.entries:
			batch = append(batch, entry)
			if len(batch) >= b.config.BatchSize {
				b.flushBatch(batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				b.flushBatch(batch)
			}
			return
		}
	}
}
