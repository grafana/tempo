package ingester

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/cortexproject/cortex/pkg/util/services"
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

const (
	// Backoff for retrying 'immediate' flushes. Only counts for queue
	// position, not wallclock time.
	flushBackoff = 1 * time.Second
)

const (
	opKindComplete = iota
	opKindFlush
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
	go func() {
		// lifecycler should exit the ring on shutdown
		i.lifecycler.SetUnregisterOnShutdown(true)

		// stop accepting new writes
		i.markUnavailable()

		// move all data into flushQueue
		i.sweepAllInstances(true)

		for !i.flushQueues.IsEmpty() {
			time.Sleep(100 * time.Millisecond)
		}

		// stop ingester service
		_ = services.StopAndAwaitTerminated(context.Background(), i)
	}()

	_, _ = w.Write([]byte("shutdown job acknowledged"))
}

// FlushHandler calls sweepAllInstances(true) which will force push all traces into the WAL and force
//  mark all head blocks as ready to flush.
func (i *Ingester) FlushHandler(w http.ResponseWriter, _ *http.Request) {
	i.sweepAllInstances(true)
	w.WriteHeader(http.StatusNoContent)
}

type flushOp struct {
	kind    int
	from    int64
	userID  string
	blockID uuid.UUID
}

func (o *flushOp) Key() string {
	return o.userID + "/" + strconv.Itoa(o.kind) + "/" + o.blockID.String()
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
	blockID, err := instance.CutBlockIfReady(i.cfg.MaxBlockDuration, i.cfg.MaxBlockBytes, immediate)
	if err != nil {
		level.Error(log.WithUserID(instance.instanceID, log.Logger)).Log("msg", "failed to cut block", "err", err)
		return
	}

	if blockID != uuid.Nil {
		i.flushQueues.Enqueue(&flushOp{
			kind:    opKindComplete,
			from:    math.MaxInt64,
			userID:  instance.instanceID,
			blockID: blockID,
		})
	}

	// dump any blocks that have been flushed for awhile
	err = instance.ClearFlushedBlocks(i.cfg.CompleteBlockTimeout)
	if err != nil {
		level.Error(log.WithUserID(instance.instanceID, log.Logger)).Log("msg", "failed to complete block", "err", err)
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

		var completeBlockID uuid.UUID
		var err error
		if op.kind == opKindComplete {
			level.Debug(log.Logger).Log("msg", "completing block", "userid", op.userID)
			instance, exists := i.getInstanceByID(op.userID)
			if !exists {
				// instance no longer exists? that's bad, log and continue
				level.Error(log.Logger).Log("msg", "instance not found", "tenantID", op.userID)
				continue
			}

			completeBlockID, err = instance.CompleteBlock(op.blockID)
			if completeBlockID != uuid.Nil {
				// add a flushOp for the block we just completed
				i.flushQueues.Enqueue(&flushOp{
					kind:    opKindFlush,
					from:    time.Now().Unix(),
					userID:  instance.instanceID,
					blockID: completeBlockID,
				})
			}

		} else {
			level.Debug(log.Logger).Log("msg", "flushing block", "userid", op.userID, "fp")

			err = i.flushBlock(op.userID, op.blockID)
		}

		if err != nil {
			level.Error(log.WithUserID(op.userID, log.Logger)).Log("msg", "error performing op in flushQueue",
				"op", op.kind, "block", op.blockID.String(), "err", err)
			// re-queue op with backoff
			op.from += int64(flushBackoff)
			i.flushQueues.Requeue(op)
			continue
		}

		i.flushQueues.Clear(op)
	}
}

func (i *Ingester) flushBlock(userID string, blockID uuid.UUID) error {
	instance, err := i.getOrCreateInstance(userID)
	if err != nil {
		return err
	}

	if instance == nil {
		return fmt.Errorf("instance id %s not found", userID)
	}

	if block := instance.GetBlockToBeFlushed(blockID); block != nil {
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
