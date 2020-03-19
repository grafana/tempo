package ingester

import (
	"context"
	"fmt"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
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

type traceMapShard struct {
	tracesMtx *sync.RWMutex
	traces    map[traceFingerprint]*trace
}

type instance struct {
	traceMapShardMtx *sync.RWMutex
	traceMapShards   map[string]*traceMapShard

	blockTracesMtx sync.RWMutex
	headBlock      *tempodb_wal.HeadBlock
	completeBlocks []*tempodb_wal.CompleteBlock
	lastBlockCut   time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	tracesInMemory     uint64
	limiter            *Limiter
	wal                *tempodb_wal.WAL
}

func newInstance(instanceID string, limiter *Limiter, wal *tempodb_wal.WAL) (*instance, error) {
	i := &instance{
		traceMapShardMtx: new(sync.RWMutex),
		traceMapShards:   make(map[string]*traceMapShard, 256),

		instanceID:         instanceID,
		tracesCreatedTotal: metricTracesCreatedTotal.WithLabelValues(instanceID),
		limiter:            limiter,
		wal:                wal,
		seed:               maphash.MakeSeed(),
	}
	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *instance) getOrCreateShard(ctx context.Context, traceID []byte) (*traceMapShard, bool) {
	// Hash to evenly distribute search space
	hasher := maphash.Hash{}
	hasher.SetSeed(i.seed)
	hasher.Write(traceID)
	idSum := []byte{}
	shardKey := fmt.Sprintf("%x", hasher.Sum(idSum))[0:1]

	// Check if shard exists
	// After a first few requests this should always be true
	i.traceMapShardMtx.RLock()
	shard, ok := i.traceMapShards[shardKey]
	i.traceMapShardMtx.RUnlock()

	if ok {
		return shard, false
	}

	// Create a shard. Need a write lock here
	i.traceMapShardMtx.Lock()
	defer i.traceMapShardMtx.Unlock()

	// Shard might've been created by another process
	shard, ok = i.traceMapShards[shardKey]
	if ok {
		return shard, false
	}

	shard = &traceMapShard{
		tracesMtx: new(sync.RWMutex),
		traces:    make(map[traceFingerprint]*trace),
	}
	i.traceMapShards[shardKey] = shard
	return shard, true
}

func (i *instance) Push(ctx context.Context, req *tempopb.PushRequest) error {
	trace, err := i.getOrCreateTrace(ctx, req)
	if err != nil {
		return err
	}

	if trace == nil {
		return errors.New("Error creating trace")
	}
	if err := trace.Push(ctx, req); err != nil {
		return err
	}

	return nil
}

// PushBytes is used by the wal replay code and so it can push directly into the head block with 0 shenanigans
func (i *instance) PushBytes(ctx context.Context, id tempodb_backend.ID, object []byte) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	return i.headBlock.Write(id, object)
}

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	now := time.Now()
	for _, shard := range i.traceMapShards {
		shard.tracesMtx.Lock()
		for key, trace := range shard.traces {
			if now.Add(cutoff).After(trace.lastAppend) || immediate {
				out, err := proto.Marshal(trace.trace)
				if err != nil {
					return err
				}

				err = i.headBlock.Write(trace.traceID, out)
				if err != nil {
					return err
				}

				delete(shard.traces, key)
				// Subtract one from numTracesOnMap per https://godoc.org/sync/atomic#AddUint64
				atomic.AddUint64(&i.tracesInMemory, ^uint64(0))
			}
		}
		shard.tracesMtx.Unlock()
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
			err = b.Clear()
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
	// Find shard
	hasher := maphash.Hash{}
	hasher.SetSeed(i.seed)
	hasher.Write(id)
	idSum := []byte{}
	shard, ok := i.traceMapShards[fmt.Sprintf("%x", hasher.Sum(idSum))[0:1]]
	if ok {
		shard.tracesMtx.Lock()
		if liveTrace, ok := shard.traces[traceFingerprint(util.Fingerprint(id))]; ok {
			retMe := liveTrace.trace
			shard.tracesMtx.Unlock()
			return retMe, nil
		}
		shard.tracesMtx.Unlock()
	}

	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	foundBytes, err := i.headBlock.Find(id)
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
		foundBytes, err = c.Find(id)
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
	if len(req.Batch.Spans) == 0 {
		return nil, fmt.Errorf("invalid request received with 0 spans")
	}

	// two assumptions here should hold.  distributor separates spans by traceid.  0 length span slices should be filtered before here
	traceID := req.Batch.Spans[0].TraceId
	fp := traceFingerprint(util.Fingerprint(traceID))

	shard, _ := i.getOrCreateShard(ctx, traceID)
	shard.tracesMtx.RLock()

	trace, ok := shard.traces[fp]
	if ok {
		shard.tracesMtx.RUnlock()
		return trace, nil
	}
	shard.tracesMtx.RUnlock()

	// err := i.limiter.AssertMaxTracesPerUser(i.instanceID, int(i.tracesInMemory))
	// if err != nil {
	// 	return nil, httpgrpc.Errorf(http.StatusTooManyRequests, err.Error())
	// }

	trace = newTrace(fp, traceID)
	shard.tracesMtx.Lock()
	shard.traces[fp] = trace
	shard.tracesMtx.Unlock()

	// atomic.AddUint64(&i.tracesInMemory, 1)
	// i.tracesCreatedTotal.Inc()

	return trace, nil
}

func (i *instance) resetHeadBlock() error {
	var err error
	i.headBlock, err = i.wal.NewBlock(uuid.New(), i.instanceID)
	i.lastBlockCut = time.Now()
	return err
}
