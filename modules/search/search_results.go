package search

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
	"go.uber.org/atomic"
)

type SearchResults struct {
	resultsCh   chan *tempopb.TraceSearchMetadata
	doneCh      chan struct{}
	workerCount atomic.Int32

	tracesInspected atomic.Uint32
	bytesInspected  atomic.Uint64
}

func NewSearchResults() *SearchResults {
	return &SearchResults{
		resultsCh: make(chan *tempopb.TraceSearchMetadata),
		doneCh:    make(chan struct{}),
	}
}

func (sr *SearchResults) AddResult(ctx context.Context, r *tempopb.TraceSearchMetadata) (quit bool) {
	select {
	case sr.resultsCh <- r:
		return false
	case <-ctx.Done():
		return true
	case <-sr.doneCh:
		return true
	}
}

func (sr *SearchResults) Results() <-chan *tempopb.TraceSearchMetadata {
	return sr.resultsCh
}

func (sr *SearchResults) Close() {
	close(sr.doneCh)
}

func (sr *SearchResults) StartWorker() {
	sr.workerCount.Inc()
}

func (sr *SearchResults) FinishWorker() {
	newCount := sr.workerCount.Dec()
	if newCount == 0 {
		close(sr.resultsCh)
	}
}

func (sr *SearchResults) TracesInspected() uint32 {
	return sr.tracesInspected.Load()
}

func (sr *SearchResults) AddTraceInspected() {
	sr.tracesInspected.Inc()
}

func (sr *SearchResults) BytesInspected() uint64 {
	return sr.bytesInspected.Load()
}

func (sr *SearchResults) AddBytesInspected(c uint64) {
	sr.bytesInspected.Add(c)
}
