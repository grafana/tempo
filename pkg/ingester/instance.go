package ingester

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	tempodb_encoding "github.com/grafana/tempo/tempodb/encoding"
	tempodb_wal "github.com/grafana/tempo/tempodb/wal"
)

type traceFingerprint uint64

// Errors returned on Query.
var (
	ErrTraceMissing = errors.New("Trace missing")
)

var (
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricBlocksClearedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	})
)

type instance struct {
	tracesMtx sync.Mutex
	traces    map[traceFingerprint]*trace

	blockTracesMtx sync.RWMutex
	headBlock      *tempodb_wal.HeadBlock
	completeBlocks []*tempodb_wal.CompleteBlock
	lastBlockCut   time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	limiter            *Limiter
	wal                *tempodb_wal.WAL
}

func newInstance(instanceID string, limiter *Limiter, wal *tempodb_wal.WAL) (*instance, error) {
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

func (i *instance) Push(ctx context.Context, req *tempopb.PushRequest) error {
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

// PushBytes is used by the wal replay code and so it can push directly into the head block with 0 shenanigans
func (i *instance) PushBytes(ctx context.Context, id tempodb_encoding.ID, object []byte) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	return i.headBlock.Write(id, object)
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
			out, err := proto.Marshal(trace.trace)
			if err != nil {
				return err
			}

			err = i.headBlock.Write(trace.traceID, out)
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
		completeBlock, err := i.headBlock.Complete(i.wal, i)
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

func (i *instance) GetBlockToBeFlushed() *tempodb_wal.CompleteBlock {
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
			err = b.Clear() // todo: don't remove from complete blocks slice until after clear succeeds?
			if err == nil {
				metricBlocksClearedTotal.Inc()
			}
			break
		}
	}

	return err
}

func (i *instance) FindTraceByID(id []byte) (*tempopb.Trace, error) {
	// First search live traces being assembled in the ingester instance.
	i.tracesMtx.Lock()
	if liveTrace, ok := i.traces[traceFingerprint(util.Fingerprint(id))]; ok {
		retMe := liveTrace.trace // todo: is this necessary?
		i.tracesMtx.Unlock()
		return retMe, nil
	}
	i.tracesMtx.Unlock()

	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	foundBytes, err := i.headBlock.Find(id, i)
	if err != nil {
		return nil, err
	}
	if foundBytes != nil {
		out := &tempopb.Trace{}

		err = proto.Unmarshal(foundBytes, out)
		if err != nil {
			return nil, err
		}

		return out, nil
	}

	for _, c := range i.completeBlocks {
		foundBytes, err = c.Find(id, i)
		if err != nil {
			return nil, err
		}
		if foundBytes != nil {
			out := &tempopb.Trace{}

			err = proto.Unmarshal(foundBytes, out)
			if err != nil {
				return nil, err
			}
			return out, err
		}
	}

	return nil, nil
}

func (i *instance) getOrCreateTrace(req *tempopb.PushRequest) (*trace, error) {
	traceID, err := pushRequestTraceID(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get trace id %w", err)
	}

	fp := traceFingerprint(util.Fingerprint(traceID))

	trace, ok := i.traces[fp]
	if ok {
		return trace, nil
	}

	err = i.limiter.AssertMaxTracesPerUser(i.instanceID, len(i.traces))
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

func (i *instance) Combine(objA []byte, objB []byte) []byte {
	return util.CombineTraces(objA, objB)
}

// pushRequestTraceID gets the TraceID of the first span in the batch and assumes its the trace ID throughout
//  this assumption should hold b/c the distributors make sure each batch all belong to the same trace
func pushRequestTraceID(req *tempopb.PushRequest) ([]byte, error) {
	if req == nil || req.Batch == nil {
		return nil, errors.New("req or req.Batch nil")
	}

	if len(req.Batch.InstrumentationLibrarySpans) == 0 {
		return nil, errors.New("InstrumentationLibrarySpans has length 0")
	}

	if len(req.Batch.InstrumentationLibrarySpans[0].Spans) == 0 {
		return nil, errors.New("Spans has length 0")
	}

	return req.Batch.InstrumentationLibrarySpans[0].Spans[0].TraceId, nil
}
