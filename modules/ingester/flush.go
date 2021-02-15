package ingester

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/grafana/tempo/tempodb/wal"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
)

var (
	metricBlocksFlushed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_blocks_flushed_total",
		Help:      "The total number of blocks flushed",
	})
	metricFailedFlushes = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_failed_flushes_total",
		Help:      "The total number of failed traces",
	})
	metricFlushDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "ingester_flush_duration_seconds",
		Help:      "Records the amount of time to flush a complete block.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	})
)

type opKind int

const (
	// Backoff for retrying 'immediate' flushes. Only counts for queue
	// position, not wallclock time.
	flushBackoff = 1 * time.Second

	complete opKind = iota
	flush    opKind = iota
)

// Flush triggers a flush of all in memory traces to disk.  This is called
// by the lifecycler on shutdown and will put our traces in the WAL to be
// replayed.
func (i *Ingester) Flush() {
	instances := i.getInstances()

	for _, instance := range instances {
		err := instance.CutCompleteTraces(0, true)
		if err != nil {
			level.Error(log.WithUserID(instance.instanceID, log.Logger)).Log("msg", "failed to cut complete traces on shutdown", "err", err)
		}
	}
}

// ShutdownHandler handles a graceful shutdown for an ingester. It does the following things in order
// * Stop incoming writes by exiting from the ring
// * Flush all blocks to backend
func (i *Ingester) ShutdownHandler(w http.ResponseWriter, _ *http.Request) {
	// stop accepting new writes
	err := i.markUnavailable()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error marking ingester unavailable"))
	}

	// move all data into flushQueue
	i.sweepAllInstances(true)

	// lifecycler should exit the ring on shutdown
	i.lifecycler.SetUnregisterOnShutdown(true)

	// stop ingester which will internally stop lifecycler
	_ = services.StopAndAwaitTerminated(context.Background(), i)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ingester successfully shutdown"))
}

// FlushHandler calls sweepAllInstances(true) which will force push all traces into the WAL and force
//  mark all head blocks as ready to flush.
func (i *Ingester) FlushHandler(w http.ResponseWriter, _ *http.Request) {
	i.sweepAllInstances(true)
	w.WriteHeader(http.StatusNoContent)
}

type flushOp struct {
	kind            opKind
	from            int64
	userID          string
	completingBlock *wal.AppendBlock
}

func (o *flushOp) Key() string {
	return o.userID
}

func (o *flushOp) Priority() int64 {
	return -o.from
}

// sweepAllInstances periodically schedules series for flushing and garbage collects instances with no series
func (i *Ingester) sweepAllInstances(immediate bool) {
	instances := i.getInstances()

	for _, instance := range instances {
		i.sweepInstance(instance, immediate)
	}
}

func (i *Ingester) sweepInstance(instance *instance, immediate bool) {
	// cut traces internally
	err := instance.CutCompleteTraces(i.cfg.MaxTraceIdle, immediate)
	if err != nil {
		level.Error(log.WithUserID(instance.instanceID, log.Logger)).Log("msg", "failed to cut traces", "err", err)
		return
	}

	// see if it's ready to cut a block
	completingBlock, err := instance.CutBlockIfReady(i.cfg.MaxBlockDuration, i.cfg.MaxBlockBytes, immediate)
	if err != nil {
		level.Error(log.WithUserID(instance.instanceID, log.Logger)).Log("msg", "failed to cut block", "err", err)
		return
	}

	// enqueue completingBlock if not nil
	if completingBlock != nil {
		instance.waitForFlush.Inc()
		i.flushQueues.Enqueue(&flushOp{
			kind:            complete,
			completingBlock: completingBlock,
			from:            math.MaxInt64,
			userID:          instance.instanceID,
		})
	}

	// dump any blocks that have been flushed for awhile
	err = instance.ClearFlushedBlocks(i.cfg.CompleteBlockTimeout)
	if err != nil {
		level.Error(log.WithUserID(instance.instanceID, log.Logger)).Log("msg", "failed to complete block", "err", err)
	}

	// need a way to check that all completingBlocks have been flushed...
	if immediate {
		for instance.waitForFlush.Load() != 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// see if any complete blocks are ready to be flushed
	// these might get double queued if a routine flush coincides with a shutdown .. but that's OK.
	for range instance.GetBlocksToBeFlushed() {
		i.flushQueues.Enqueue(&flushOp{
			kind:   flush,
			from:   time.Now().Unix(),
			userID: instance.instanceID,
		})
	}
}

func (i *Ingester) flushLoop(j int) {
	defer func() {
		level.Debug(log.Logger).Log("msg", "Ingester.flushLoop() exited")
		i.flushQueuesDone.Done()
	}()

	for {
		o := i.flushQueues.Dequeue(j)
		if o == nil {
			return
		}
		op := o.(*flushOp)

		if op.kind == complete {
			level.Debug(log.Logger).Log("msg", "completing block", "userid", op.userID, "fp")
			instance, exists := i.getInstanceByID(op.userID)
			if !exists {
				// instance no longer exists? that's bad, clear and continue
				_ = op.completingBlock.Clear()
				continue
			}

			completeBlock, err := instance.writer.CompleteBlock(op.completingBlock, instance)

			instance.blocksMtx.Lock()
			if err != nil {
				// this is a really bad error that results in data loss.  most likely due to disk full
				_ = op.completingBlock.Clear()
				metricFailedFlushes.Inc()
				level.Error(log.Logger).Log("msg", "unable to complete block.  THIS BLOCK WAS LOST", "tenantID", op.userID, "err", err)
				instance.blocksMtx.Unlock()
				continue
			}
			instance.completeBlocks = append(instance.completeBlocks, completeBlock)
			instance.blocksMtx.Unlock()
		} else {
			level.Debug(log.Logger).Log("msg", "flushing block", "userid", op.userID, "fp")

			err := i.flushUserTraces(op.userID)
			if err != nil {
				level.Error(log.WithUserID(op.userID, log.Logger)).Log("msg", "failed to flush user", "err", err)

				// re-queue failed flush
				op.from += int64(flushBackoff)
				i.flushQueues.Requeue(op)
				continue
			}

			instance, exists := i.getInstanceByID(op.userID)
			if !exists {
				continue
			}
			instance.waitForFlush.Dec()
		}

		i.flushQueues.Clear(op)
	}
}

func (i *Ingester) flushUserTraces(userID string) error {
	instance, err := i.getOrCreateInstance(userID)
	if err != nil {
		return err
	}

	if instance == nil {
		return fmt.Errorf("instance id %s not found", userID)
	}

	for _, block := range instance.GetBlocksToBeFlushed() {
		ctx := user.InjectOrgID(context.Background(), userID)
		ctx, cancel := context.WithTimeout(ctx, i.cfg.FlushOpTimeout)
		defer cancel()

		start := time.Now()
		err = i.store.WriteBlock(ctx, block)
		metricFlushDuration.Observe(time.Since(start).Seconds())
		if err != nil {
			metricFailedFlushes.Inc()
			return err
		}
		metricBlocksFlushed.Inc()
	}

	return nil
}
