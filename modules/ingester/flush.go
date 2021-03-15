package ingester

import (
	"context"
	"fmt"
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
	initialBackoff      = 1 * time.Second
	maxBackoff          = time.Minute
	maxCompleteAttempts = 10
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
		level.Info(log.Logger).Log("msg", "shutdown handler called")

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

		level.Info(log.Logger).Log("msg", "shutdown handler complete")
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
	kind     int
	at       time.Time // When to execute
	attempts uint
	backoff  time.Duration
	userID   string
	blockID  uuid.UUID
}

func (o *flushOp) Key() string {
	return o.userID + "/" + strconv.Itoa(o.kind) + "/" + o.blockID.String()
}

// Priority orders entries in the queue. The larger the number the higher the priority, so inverted here to
// prioritize entries with earliest timestamps.
func (o *flushOp) Priority() int64 {
	return -o.at.Unix()
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
			at:      time.Now(),
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
		op.attempts++

		retry := false

		if op.kind == opKindComplete {
			level.Debug(log.Logger).Log("msg", "completing block", "userid", op.userID)
			instance, exists := i.getInstanceByID(op.userID)
			if !exists {
				// instance no longer exists? that's bad, log and continue
				level.Error(log.Logger).Log("msg", "instance not found", "userID", op.userID, "block", op.blockID.String())
				continue
			}

			err := instance.CompleteBlock(op.blockID)
			if err != nil {
				handleFlushError(op, err)

				if op.attempts >= maxCompleteAttempts {
					level.Error(log.WithUserID(op.userID, log.Logger)).Log("msg", "Block exceeded max completion errors. Deleting. POSSIBLE DATA LOSS",
						"userID", op.userID, "attempts", op.attempts, "block", op.blockID.String())
					instance.ClearCompletingBlock(op.blockID)
				} else {
					retry = true
				}
			} else {
				// add a flushOp for the block we just completed
				i.flushQueues.Enqueue(&flushOp{
					kind:    opKindFlush,
					at:      time.Now(),
					userID:  instance.instanceID,
					blockID: op.blockID,
				})
			}

		} else {
			level.Info(log.Logger).Log("msg", "flushing block", "userid", op.userID, "block", op.blockID.String())

			err := i.flushBlock(op.userID, op.blockID)
			if err != nil {
				handleFlushError(op, err)
				retry = true
			}
		}

		if retry {
			i.requeue(op)
		} else {
			i.flushQueues.Clear(op)
		}
	}
}

func handleFlushError(op *flushOp, err error) {
	level.Error(log.WithUserID(op.userID, log.Logger)).Log("msg", "error performing op in flushQueue",
		"op", op.kind, "block", op.blockID.String(), "attempts", op.attempts, "err", err)
	metricFailedFlushes.Inc()
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
			return err
		}
		// Delete original wal only after successful flush
		instance.ClearCompletingBlock(blockID)
		metricBlocksFlushed.Inc()
	} else {
		return fmt.Errorf("error getting block to flush")
	}

	return nil
}

func (i *Ingester) requeue(op *flushOp) {
	op.backoff *= 2
	if op.backoff < initialBackoff {
		op.backoff = initialBackoff
	}
	if op.backoff > maxBackoff {
		op.backoff = maxBackoff
	}

	op.at = time.Now().Add(op.backoff)

	level.Info(log.WithUserID(op.userID, log.Logger)).Log("msg", "retrying op in flushQueue",
		"op", op.kind, "block", op.blockID.String(), "backoff", op.backoff)

	go func() {
		time.Sleep(op.backoff)
		i.flushQueues.Requeue(op)
	}()
}
