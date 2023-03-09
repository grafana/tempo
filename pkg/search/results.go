package search

import (
	"context"
	"sync"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"go.uber.org/atomic"
)

// Results eases performing a highly parallel search by funneling all results into a single
// channel that is easy to consume, signaling workers to quit early as needed, and collecting
// metrics.
type Results struct {
	resultsCh chan *tempopb.TraceSearchMetadata
	doneCh    chan struct{}
	quit      atomic.Bool
	error     atomic.Error
	started   atomic.Bool
	wg        sync.WaitGroup

	tracesInspected atomic.Uint32
	bytesInspected  atomic.Uint64
	blocksInspected atomic.Uint32
	blocksSkipped   atomic.Uint32
}

func NewResults() *Results {
	return &Results{
		resultsCh: make(chan *tempopb.TraceSearchMetadata),
		doneCh:    make(chan struct{}),
	}
}

// AddResult sends a search result from a search task (goroutine) to the receiver of the
// search results, i.e. the initiator of the search.  This function blocks until there
// is buffer space in the results channel or if the task should stop searching because the
// receiver went away or the given context is done. In this case true is returned.
func (sr *Results) AddResult(ctx context.Context, r *tempopb.TraceSearchMetadata) (quit bool) {
	if sr.quit.Load() {
		return true
	}

	select {
	case sr.resultsCh <- r:
		return false
	case <-ctx.Done():
		return true
	case <-sr.doneCh:
		// This returns immediately once the done channel is closed.
		return true
	}
}

// SetError will set error in a thread safe manner.
//
// NOTE: this will ignore common.Unsupported errors,
// we don't Propagate those error upstream.
func (sr *Results) SetError(err error) {
	if err != common.ErrUnsupported { // ignore common.Unsupported
		sr.error.Store(err)
	}
}

// Error returns error set by SetError
func (sr *Results) Error() error {
	return sr.error.Load()
}

// Quit returns if search tasks should quit early. This can occur due to max results
// already found, or other errors such as timeout, etc.
func (sr *Results) Quit() bool {
	return sr.quit.Load()
}

// Results returns the results channel. Channel is closed when the search is complete.
// Can be iterated by range like:
// for res := range sr.Results()
func (sr *Results) Results() <-chan *tempopb.TraceSearchMetadata {
	return sr.resultsCh
}

// Close signals to all workers to quit, when max results is received and no more work is needed.
// Called by the initiator of the search in a defer statement like:
// sr := NewSearchResults()
// defer sr.Close()
func (sr *Results) Close() {
	// Only once
	if sr.quit.CompareAndSwap(false, true) {
		// Closing done channel makes all subsequent and blocked calls to AddResult return
		// quit immediately.
		close(sr.doneCh)
	}
}

// StartWorker indicates another sender will be using the results channel. Must be followed
// with a call to FinishWorker which is usually deferred in a goroutine:
// sr.StartWorker()
// go func() {
// defer sr.FinishWorker()
func (sr *Results) StartWorker() {
	sr.wg.Add(1)
}

// AllWorkersStarted indicates that no more workers (senders) will be launched, and the
// results channel can be closed once the number of workers reaches zero.  This function
// call occurs after all calls to StartWorker.
func (sr *Results) AllWorkersStarted() {
	// Only once
	if sr.started.CompareAndSwap(false, true) {
		// Close results when all workers finished.
		go func() {
			sr.wg.Wait()
			close(sr.resultsCh)
		}()
	}
}

// FinishWorker indicates a sender (goroutine) is done searching and will not
// send any more search results. When the last sender is finished, the results
// channel is closed.
func (sr *Results) FinishWorker() {
	sr.wg.Add(-1)
}

func (sr *Results) TracesInspected() uint32 {
	return sr.tracesInspected.Load()
}

func (sr *Results) AddTraceInspected(c uint32) {
	sr.tracesInspected.Add(c)
}

func (sr *Results) BytesInspected() uint64 {
	return sr.bytesInspected.Load()
}

func (sr *Results) AddBytesInspected(c uint64) {
	sr.bytesInspected.Add(c)
}

func (sr *Results) AddBlockInspected() {
	sr.blocksInspected.Inc()
}

func (sr *Results) BlocksInspected() uint32 {
	return sr.blocksInspected.Load()
}

func (sr *Results) AddBlockSkipped() {
	sr.blocksSkipped.Inc()
}

func (sr *Results) BlocksSkipped() uint32 {
	return sr.blocksSkipped.Load()
}
