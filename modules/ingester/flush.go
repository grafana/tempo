package ingester

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	gklog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	ot "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/uber/jaeger-client-go"

	"github.com/grafana/tempo/pkg/util/log"
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
	metricFlushRetries = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_flush_retries_total",
		Help:      "The total number of retries after a failed flush",
	})
	metricFlushFailedRetries = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_flush_failed_retries_total",
		Help:      "The total number of failed retries after a failed flush",
	})
	metricFlushDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "ingester_flush_duration_seconds",
		Help:      "Records the amount of time to flush a complete block.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	})
	metricFlushSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "ingester_flush_size_bytes",
		Help:      "Size in bytes of blocks flushed.",
		Buckets:   prometheus.ExponentialBuckets(1024*1024, 2, 10), // from 1MB up to 1GB
	})
)

const (
	initialBackoff      = 30 * time.Second
	flushJitter         = 10 * time.Second
	maxBackoff          = 120 * time.Second
	maxCompleteAttempts = 3
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

		// flush any remaining traces
		i.flushRemaining()

		// stop ingester service
		_ = services.StopAndAwaitTerminated(context.Background(), i)

		level.Info(log.Logger).Log("msg", "shutdown handler complete")
	}()

	_, _ = w.Write([]byte("shutdown job acknowledged"))
}

// FlushHandler calls sweepAllInstances(true) which will force push all traces into the WAL and force
// mark all head blocks as ready to flush. It will either flush all instances or if an instance is specified,
// just that one.
func (i *Ingester) FlushHandler(w http.ResponseWriter, r *http.Request) {
	queryParamInstance := "tenant"

	if r.URL.Query().Has(queryParamInstance) {
		instance, ok := i.getInstanceByID(r.URL.Query().Get(queryParamInstance))
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		level.Info(log.Logger).Log("msg", "flushing instance", "instance", instance.instanceID)
		i.sweepInstance(instance, true)
	} else {
		i.sweepAllInstances(true)
	}

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
		level.Info(log.Logger).Log("msg", "head block cut. enqueueing flush op", "userid", instance.instanceID, "block", blockID)
		// jitter to help when flushing many instances at the same time
		// no jitter if immediate (initiated via /flush handler for example)
		i.enqueue(&flushOp{
			kind:    opKindComplete,
			userID:  instance.instanceID,
			blockID: blockID,
		}, !immediate)
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

		var retry bool
		var err error

		if op.kind == opKindComplete {
			retry, err = i.handleComplete(op)
		} else {
			retry, err = i.handleFlush(context.Background(), op.userID, op.blockID)
		}

		if err != nil {
			handleFailedOp(op, err)
		}

		if retry {
			i.requeue(op)
		} else {
			i.flushQueues.Clear(op)
		}
	}
}

func handleFailedOp(op *flushOp, err error) {
	level.Error(log.WithUserID(op.userID, log.Logger)).Log("msg", "error performing op in flushQueue",
		"op", op.kind, "block", op.blockID.String(), "attempts", op.attempts, "err", err)
	metricFailedFlushes.Inc()

	if op.attempts > 1 {
		metricFlushFailedRetries.Inc()
	}
}

func handleAbandonedOp(op *flushOp) {
	level.Info(log.WithUserID(op.userID, log.Logger)).Log("msg", "Abandoning op in flush queue because ingester is shutting down",
		"op", op.kind, "block", op.blockID.String(), "attempts", op.attempts)
}

func (i *Ingester) handleComplete(op *flushOp) (retry bool, err error) {
	// No point in proceeding if shutdown has been initiated since
	// we won't be able to queue up the next flush op
	if i.flushQueues.IsStopped() {
		handleAbandonedOp(op)
		return false, nil
	}

	start := time.Now()
	level.Info(log.Logger).Log("msg", "completing block", "userid", op.userID, "blockID", op.blockID)
	instance, err := i.getOrCreateInstance(op.userID)
	if err != nil {
		return false, err
	}

	err = instance.CompleteBlock(op.blockID)
	level.Info(log.Logger).Log("msg", "block completed", "userid", op.userID, "blockID", op.blockID, "duration", time.Since(start))
	if err != nil {
		handleFailedOp(op, err)

		if op.attempts >= maxCompleteAttempts {
			level.Error(log.WithUserID(op.userID, log.Logger)).Log("msg", "Block exceeded max completion errors. Deleting. POSSIBLE DATA LOSS",
				"userID", op.userID, "attempts", op.attempts, "block", op.blockID.String())

			// Delete WAL and move on
			err = instance.ClearCompletingBlock(op.blockID)
			return false, err
		}

		return true, nil
	}

	err = instance.ClearCompletingBlock(op.blockID)
	if err != nil {
		return false, fmt.Errorf("error clearing completing block: %w", err)
	}

	// add a flushOp for the block we just completed
	// No delay
	i.enqueue(&flushOp{
		kind:    opKindFlush,
		userID:  instance.instanceID,
		blockID: op.blockID,
	}, false)

	return false, nil
}

// withSpan adds traceID to a logger, if span is sampled
// TODO: move into some central trace/log package
func withSpan(logger gklog.Logger, sp ot.Span) gklog.Logger {
	if sp == nil {
		return logger
	}
	sctx, ok := sp.Context().(jaeger.SpanContext)
	if !ok || !sctx.IsSampled() {
		return logger
	}

	return gklog.With(logger, "traceID", sctx.TraceID().String())
}

func (i *Ingester) handleFlush(ctx context.Context, userID string, blockID uuid.UUID) (retry bool, err error) {
	sp, ctx := ot.StartSpanFromContext(ctx, "flush", ot.Tag{Key: "organization", Value: userID}, ot.Tag{Key: "blockID", Value: blockID.String()})
	defer sp.Finish()
	withSpan(level.Info(log.Logger), sp).Log("msg", "flushing block", "userid", userID, "block", blockID.String())

	instance, err := i.getOrCreateInstance(userID)
	if err != nil {
		return true, err
	}

	if instance == nil {
		return false, fmt.Errorf("instance id %s not found", userID)
	}

	if block := instance.GetBlockToBeFlushed(blockID); block != nil {
		ctx := user.InjectOrgID(ctx, userID)
		ctx, cancel := context.WithTimeout(ctx, i.cfg.FlushOpTimeout)
		defer cancel()

		start := time.Now()
		err = i.store.WriteBlock(ctx, block)
		metricFlushDuration.Observe(time.Since(start).Seconds())
		metricFlushSize.Observe(float64(block.BlockMeta().Size))
		if err != nil {
			ext.Error.Set(sp, true)
			sp.LogFields(otlog.Error(err))
			return true, err
		}

		metricBlocksFlushed.Inc()
	} else {
		return false, fmt.Errorf("error getting block to flush")
	}

	return false, nil
}

func (i *Ingester) enqueue(op *flushOp, jitter bool) {
	delay := time.Duration(0)

	if jitter {
		delay = time.Duration(rand.Float32() * float32(flushJitter))
	}

	op.at = time.Now().Add(delay)

	go func() {
		time.Sleep(delay)

		// Check if shutdown initiated
		if i.flushQueues.IsStopped() {
			handleAbandonedOp(op)
			return
		}

		err := i.flushQueues.Enqueue(op)
		if err != nil {
			handleFailedOp(op, err)
		}
	}()
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

		// Check if shutdown initiated
		if i.flushQueues.IsStopped() {
			handleAbandonedOp(op)
			return
		}

		metricFlushRetries.Inc()

		err := i.flushQueues.Requeue(op)
		if err != nil {
			handleFailedOp(op, err)
		}
	}()
}
