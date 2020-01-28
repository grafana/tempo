package ingester

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/frigg/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
)

var (
	tracesFlushed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "ingester_traces_flushed_total",
		Help:      "The total number of traces flushed",
	})
	failedFlushes = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "ingester_failed_flushes_total",
		Help:      "The total number of failed traces",
	})
)

const (
	// Backoff for retrying 'immediate' flushes. Only counts for queue
	// position, not wallclock time.
	flushBackoff = 1 * time.Second
)

// Flush triggers a flush of all the chunks and closes the flush queues.
// Called from the Lifecycler as part of the ingester shutdown.
func (i *Ingester) Flush() {
	i.sweepUsers(true)

	// Close the flush queues, to unblock waiting workers.
	for _, flushQueue := range i.flushQueues {
		flushQueue.Close()
	}

	i.flushQueuesDone.Wait()
}

// FlushHandler triggers a flush of all in memory chunks.  Mainly used for
// local testing.
func (i *Ingester) FlushHandler(w http.ResponseWriter, _ *http.Request) {
	i.sweepUsers(true)
	w.WriteHeader(http.StatusNoContent)
}

type flushOp struct {
	from      int64
	userID    string
	immediate bool
}

func (o *flushOp) Key() string {
	return fmt.Sprintf("%s-%v", o.userID, o.immediate)
}

func (o *flushOp) Priority() int64 {
	return -int64(o.from)
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
	ready, err := instance.CutBlockIfReady(i.cfg.MaxTracesPerBlock, i.cfg.MaxBlockDuration)
	if err != nil {
		level.Error(util.WithUserID(instance.instanceID, util.Logger)).Log("msg", "failed to cut block", "err", err)
		return
	}

	if ready {
		i.flushQueueIndex++
		flushQueueIndex := i.flushQueueIndex % i.cfg.ConcurrentFlushes
		i.flushQueues[flushQueueIndex].Enqueue(&flushOp{
			time.Now().Unix(),
			instance.instanceID,
			immediate,
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

		level.Debug(util.Logger).Log("msg", "flushing stream", "userid", op.userID, "fp", "immediate", op.immediate)

		err := i.flushUserTraces(op.userID, op.immediate)
		if err != nil {
			level.Error(util.WithUserID(op.userID, util.Logger)).Log("msg", "failed to flush user", "err", err)
		}

		if err != nil {
			op.from += int64(flushBackoff)
			i.flushQueues[j].Enqueue(op)
		}
	}
}

func (i *Ingester) flushUserTraces(userID string, immediate bool) error {
	instance, err := i.getOrCreateInstance(userID)
	if err != nil {
		return err
	}

	if instance == nil {
		return fmt.Errorf("instance id %s not found", userID)
	}

	for {
		block := instance.GetCompleteBlock()
		if block == nil {
			break
		}

		ctx := user.InjectOrgID(context.Background(), userID)
		ctx, cancel := context.WithTimeout(ctx, i.cfg.FlushOpTimeout)
		defer cancel()

		err = i.store.(storage.Store).WriteBlock(ctx, block)
		if err != nil {
			failedFlushes.Inc()
			return err
		}
		tracesFlushed.Add(float64(block.Length()))

		err = instance.ClearCompleteBlock(block)
	}

	return nil
}
