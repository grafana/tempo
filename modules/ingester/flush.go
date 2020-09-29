package ingester

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
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

// Flush triggers a flush of all in memory traces to disk.  This is called
// by the lifecycler on shutdown and will put our traces in the WAL to be
// replayed.
func (i *Ingester) Flush() {
	instances := i.getInstances()

	for _, instance := range instances {
		err := instance.CutCompleteTraces(0, true)
		if err != nil {
			level.Error(util.WithUserID(instance.instanceID, util.Logger)).Log("msg", "failed to cut complete traces on shutdown", "err", err)
		}
	}
}

// FlushHandler triggers a flush of all in memory chunks.  Mainly used for
// local testing.
func (i *Ingester) FlushHandler(w http.ResponseWriter, _ *http.Request) {
	i.sweepUsers(true)
	w.WriteHeader(http.StatusNoContent)
}

type flushOp struct {
	from   int64
	userID string
}

func (o *flushOp) Key() string {
	return o.userID
}

func (o *flushOp) Priority() int64 {
	return -o.from
}

// sweepUsers periodically schedules series for flushing and garbage collects users with no series
func (i *Ingester) sweepUsers(immediate bool) {
	instances := i.getInstances()

	for _, instance := range instances {
		i.sweepInstance(instance, immediate)
	}
}

func (i *Ingester) sweepInstance(instance *instance, immediate bool) {
	// cut traces internally
	err := instance.CutCompleteTraces(i.cfg.MaxTraceIdle, immediate)
	if err != nil {
		level.Error(util.WithUserID(instance.instanceID, util.Logger)).Log("msg", "failed to cut traces", "err", err)
		return
	}

	// see if it's ready to cut a block?
	err = instance.CutBlockIfReady(i.cfg.MaxTracesPerBlock, i.cfg.MaxBlockDuration, immediate)
	if err != nil {
		level.Error(util.WithUserID(instance.instanceID, util.Logger)).Log("msg", "failed to cut block", "err", err)
		return
	}

	// dump any blocks that have been flushed for awhile
	err = instance.ClearFlushedBlocks(i.cfg.CompleteBlockTimeout)
	if err != nil {
		level.Error(util.WithUserID(instance.instanceID, util.Logger)).Log("msg", "failed to complete block", "err", err)
	}

	// see if any complete blocks are ready to be flushed
	if instance.GetBlockToBeFlushed() != nil {
		i.flushQueueIndex++
		flushQueueIndex := i.flushQueueIndex % i.cfg.ConcurrentFlushes
		i.flushQueues[flushQueueIndex].Enqueue(&flushOp{
			time.Now().Unix(),
			instance.instanceID,
		})
	}
}

func (i *Ingester) flushLoop(j int) {
	defer func() {
		level.Debug(util.Logger).Log("msg", "Ingester.flushLoop() exited")
		i.flushQueuesDone.Done()
	}()

	for {
		o := i.flushQueues[j].Dequeue()
		if o == nil {
			return
		}
		op := o.(*flushOp)

		level.Debug(util.Logger).Log("msg", "flushing stream", "userid", op.userID, "fp")

		err := i.flushUserTraces(op.userID)
		if err != nil {
			level.Error(util.WithUserID(op.userID, util.Logger)).Log("msg", "failed to flush user", "err", err)
		}

		if err != nil {
			op.from += int64(flushBackoff)
			i.flushQueues[j].Enqueue(op)
		}
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

	for {
		block := instance.GetBlockToBeFlushed()
		if block == nil {
			break
		}

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
