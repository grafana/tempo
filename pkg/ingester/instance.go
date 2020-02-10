package ingester

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"

	"github.com/grafana/frigg/friggdb"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/util"
)

type traceFingerprint uint64

// Errors returned on Query.
var (
	ErrTraceMissing = errors.New("Trace missing")
)

var (
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricBlocksClearedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "ingester_blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	})
)

type instance struct {
	tracesMtx sync.Mutex
	traces    map[traceFingerprint]*trace

	blockTracesMtx sync.RWMutex
	headBlock      friggdb.HeadBlock
	completeBlocks []friggdb.CompleteBlock
	lastBlockCut   time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	limiter            *Limiter
	wal                friggdb.WAL
}

func newInstance(instanceID string, limiter *Limiter, wal friggdb.WAL) (*instance, error) {
	i := &instance{
		traces: map[traceFingerprint]*trace{},

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		limiter:            limiter,
		wal:                wal,
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *instance) Push(ctx context.Context, req *friggpb.PushRequest) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	trace, err := i.getOrCreateTrace(req)
	if err != nil {
		return err
	}

	if err := trace.Push(ctx, req); err != nil {
		return err
	}

	return nil
}

// PushTrace is used by the wal replay code and so it can push directly into the head block with 0 shenanigans
func (i *instance) PushTrace(ctx context.Context, t *friggpb.Trace) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	if len(t.Batches) == 0 {
		return fmt.Errorf("invalid trace received with 0 batches")
	}

	if len(t.Batches[0].Spans) == 0 {
		return fmt.Errorf("invalid batch received with 0 spans")
	}

	return i.headBlock.Write(t.Batches[0].Spans[0].TraceId, t)
}

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	now := time.Now()
	for key, trace := range i.traces {
		if now.Add(cutoff).After(trace.lastAppend) || immediate {
			err := i.headBlock.Write(trace.traceID, trace.trace)
			if err != nil {
				return err
			}

			delete(i.traces, key)
		}
	}

	return nil
}

func (i *instance) CutBlockIfReady(maxTracesPerBlock int, maxBlockLifetime time.Duration, immediate bool) (bool, error) {
	i.blockTracesMtx.RLock()
	defer i.blockTracesMtx.RUnlock()

	if i.headBlock == nil {
		return false, nil
	}

	now := time.Now()
	ready := i.headBlock.Length() >= maxTracesPerBlock || i.lastBlockCut.Add(maxBlockLifetime).Before(now) || immediate

	if ready {
		completeBlock, err := i.headBlock.Complete(i.wal)
		if err != nil {
			return false, err
		}

		i.completeBlocks = append(i.completeBlocks, completeBlock)
		err = i.resetHeadBlock()
		if err != nil {
			return false, err
		}
	}

	return ready, nil
}

func (i *instance) GetBlockToBeFlushed() friggdb.CompleteBlock {
	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	for _, c := range i.completeBlocks {
		if c.TimeWritten().IsZero() {
			return c
		}
	}

	return nil
}

func (i *instance) ClearCompleteBlocks(completeBlockTimeout time.Duration) error {
	var err error

	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	for idx, b := range i.completeBlocks {
		written := b.TimeWritten()
		if written.IsZero() {
			continue
		}

		if written.Add(completeBlockTimeout).Before(time.Now()) {
			i.completeBlocks = append(i.completeBlocks[:idx], i.completeBlocks[idx+1:]...)
			err = b.Clear()
			if err == nil {
				metricBlocksClearedTotal.Inc()
			}
			break
		}
	}

	return err
}

func (i *instance) FindTraceByID(id []byte) (*friggpb.Trace, error) {
	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	// Search live traces being assembled in the ingester instance.
	if liveTrace, ok := i.traces[traceFingerprint(util.Fingerprint(id))]; ok {
		return liveTrace.trace, nil
	}

	out := &friggpb.Trace{}

	found, err := i.headBlock.Find(id, out)
	if err != nil {
		return nil, err
	}
	if found {
		return out, nil
	}

	for _, c := range i.completeBlocks {
		found, err = c.Find(id, out)
		if err != nil {
			return nil, err
		}
		if found {
			return out, nil
		}
	}

	return nil, nil
}

func (i *instance) getOrCreateTrace(req *friggpb.PushRequest) (*trace, error) {
	if len(req.Batch.Spans) == 0 {
		return nil, fmt.Errorf("invalid request received with 0 spans")
	}

	// two assumptions here should hold.  distributor separates spans by traceid.  0 length span slices should be filtered before here
	traceID := req.Batch.Spans[0].TraceId
	fp := traceFingerprint(util.Fingerprint(traceID))

	trace, ok := i.traces[fp]
	if ok {
		return trace, nil
	}

	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, len(i.traces))
	if err != nil {
		return nil, httpgrpc.Errorf(http.StatusTooManyRequests, err.Error())
	}

	trace = newTrace(fp, traceID)
	i.traces[fp] = trace
	i.tracesCreatedTotal.Inc()

	return trace, nil
}

func (i *instance) resetHeadBlock() error {
	var err error
	i.headBlock, err = i.wal.NewBlock(uuid.New(), i.instanceID)
	i.lastBlockCut = time.Now()
	return err
}
