// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
)

// batchSender is a component that places requests into batches before passing them to the downstream senders.
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
// - concurrencyLimit is reached.
type batchSender struct {
	baseRequestSender
	cfg            exporterbatcher.Config
	mergeFunc      exporterbatcher.BatchMergeFunc[Request]
	mergeSplitFunc exporterbatcher.BatchMergeSplitFunc[Request]

	// concurrencyLimit is the maximum number of goroutines that can be created by the batcher.
	// If this number is reached and all the goroutines are busy, the batch will be sent right away.
	// Populated from the number of queue consumers if queue is enabled.
	concurrencyLimit uint64
	activeRequests   atomic.Uint64

	resetTimerCh chan struct{}

	mu          sync.Mutex
	activeBatch *batch

	logger *zap.Logger

	shutdownCh chan struct{}
	stopped    *atomic.Bool
}

// newBatchSender returns a new batch consumer component.
func newBatchSender(cfg exporterbatcher.Config, set exporter.CreateSettings) *batchSender {
	bs := &batchSender{
		activeBatch:  newEmptyBatch(),
		cfg:          cfg,
		logger:       set.Logger,
		shutdownCh:   make(chan struct{}),
		stopped:      &atomic.Bool{},
		resetTimerCh: make(chan struct{}),
	}
	return bs
}

func (bs *batchSender) Start(_ context.Context, _ component.Host) error {
	timer := time.NewTimer(bs.cfg.FlushTimeout)
	go func() {
		for {
			select {
			case <-bs.shutdownCh:
				bs.mu.Lock()
				if bs.activeBatch.request != nil {
					bs.exportActiveBatch()
				}
				bs.mu.Unlock()
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
				bs.mu.Lock()
				if bs.activeBatch.request != nil {
					bs.exportActiveBatch()
				}
				bs.mu.Unlock()
				timer.Reset(bs.cfg.FlushTimeout)
			case <-bs.resetTimerCh:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(bs.cfg.FlushTimeout)
			}
		}
	}()

	return nil
}

type batch struct {
	ctx     context.Context
	request Request
	done    chan struct{}
	err     error
}

func newEmptyBatch() *batch {
	return &batch{
		ctx:  context.Background(),
		done: make(chan struct{}),
	}
}

// exportActiveBatch exports the active batch asynchronously and replaces it with a new one.
// Caller must hold the lock.
func (bs *batchSender) exportActiveBatch() {
	go func(b *batch) {
		b.err = b.request.Export(b.ctx)
		close(b.done)
	}(bs.activeBatch)
	bs.activeBatch = newEmptyBatch()
}

// isActiveBatchReady returns true if the active batch is ready to be exported.
// The batch is ready if it has reached the minimum size or the concurrency limit is reached.
// Caller must hold the lock.
func (bs *batchSender) isActiveBatchReady() bool {
	return bs.activeBatch.request.ItemsCount() >= bs.cfg.MinSizeItems ||
		(bs.concurrencyLimit > 0 && bs.activeRequests.Load() >= bs.concurrencyLimit)
}

func (bs *batchSender) send(ctx context.Context, req Request) error {
	// Stopped batch sender should act as pass-through to allow the queue to be drained.
	if bs.stopped.Load() {
		return bs.nextSender.send(ctx, req)
	}

	if bs.cfg.MaxSizeItems > 0 {
		return bs.sendMergeSplitBatch(ctx, req)
	}
	return bs.sendMergeBatch(ctx, req)
}

// sendMergeSplitBatch sends the request to the batch which may be split into multiple requests.
func (bs *batchSender) sendMergeSplitBatch(ctx context.Context, req Request) error {
	bs.mu.Lock()
	bs.activeRequests.Add(1)
	defer bs.activeRequests.Add(^uint64(0))

	reqs, err := bs.mergeSplitFunc(ctx, bs.cfg.MaxSizeConfig, bs.activeBatch.request, req)
	if err != nil || len(reqs) == 0 {
		bs.mu.Unlock()
		return err
	}
	if len(reqs) == 1 || bs.activeBatch.request != nil {
		bs.updateActiveBatch(ctx, reqs[0])
		batch := bs.activeBatch
		if bs.isActiveBatchReady() || len(reqs) > 1 {
			bs.exportActiveBatch()
			bs.resetTimerCh <- struct{}{}
		}
		bs.mu.Unlock()
		<-batch.done
		if batch.err != nil {
			return batch.err
		}
		reqs = reqs[1:]
	} else {
		bs.mu.Unlock()
	}

	// Intentionally do not put the last request in the active batch to not block it.
	// TODO: Consider including the partial request in the error to avoid double publishing.
	for _, r := range reqs {
		if err := r.Export(ctx); err != nil {
			return err
		}
	}
	return nil
}

// sendMergeBatch sends the request to the batch and waits for the batch to be exported.
func (bs *batchSender) sendMergeBatch(ctx context.Context, req Request) error {
	bs.mu.Lock()
	bs.activeRequests.Add(1)
	defer bs.activeRequests.Add(^uint64(0))

	if bs.activeBatch.request != nil {
		var err error
		req, err = bs.mergeFunc(ctx, bs.activeBatch.request, req)
		if err != nil {
			bs.mu.Unlock()
			return err
		}
	}
	bs.updateActiveBatch(ctx, req)
	batch := bs.activeBatch
	if bs.isActiveBatchReady() {
		bs.exportActiveBatch()
		bs.resetTimerCh <- struct{}{}
	}
	bs.mu.Unlock()
	<-batch.done
	return batch.err
}

// updateActiveBatch update the active batch to the new merged request and context.
// The context is only set once and is not updated after the first call.
// Merging the context would be complex and require an additional goroutine to handle the context cancellation.
// We take the approach of using the context from the first request since it's likely to have the shortest timeout.
func (bs *batchSender) updateActiveBatch(ctx context.Context, req Request) {
	if bs.activeBatch.request == nil {
		bs.activeBatch.ctx = ctx
	}
	bs.activeBatch.request = req
}

func (bs *batchSender) Shutdown(context.Context) error {
	bs.stopped.Store(true)
	close(bs.shutdownCh)
	// Wait for the active requests to finish.
	for bs.activeRequests.Load() > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}
