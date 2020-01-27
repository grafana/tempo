package ingester

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"

	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/ingester/wal"
	"github.com/grafana/frigg/pkg/storage"
	"github.com/grafana/frigg/pkg/util"
)

type traceFingerprint uint64

const queryBatchSize = 128

// Errors returned on Query.
var (
	ErrTraceMissing = errors.New("Trace missing")
)

var (
	tracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
)

type instance struct {
	tracesMtx sync.Mutex
	traces    map[traceFingerprint]*trace

	blockTracesMtx sync.RWMutex
	traceRecords   []*storage.TraceRecord
	walBlock       wal.WALBlock
	lastBlockCut   time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	limiter            *Limiter
	wal                wal.WAL
}

func newInstance(instanceID string, limiter *Limiter, wal wal.WAL) (*instance, error) {
	i := &instance{
		traces: map[traceFingerprint]*trace{},

		instanceID:         instanceID,
		tracesCreatedTotal: tracesCreatedTotal.WithLabelValues(instanceID),
		limiter:            limiter,
		wal:                wal,
	}
	err := i.ResetBlock()
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

// Moves any complete traces out of the map to complete traces
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) error {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	now := time.Now()
	for key, trace := range i.traces {
		if now.Add(cutoff).After(trace.lastAppend) || immediate {
			start, length, err := i.walBlock.Write(trace.trace)
			if err != nil {
				return err
			}

			// insert sorted
			idx := sort.Search(len(i.traceRecords), func(idx int) bool {
				return bytes.Compare(i.traceRecords[idx].TraceID, trace.traceID) == 1
			})
			i.traceRecords = append(i.traceRecords, nil)
			copy(i.traceRecords[idx+1:], i.traceRecords[idx:])
			i.traceRecords[idx] = &storage.TraceRecord{
				TraceID: trace.traceID,
				Start:   start,
				Length:  length,
			}

			delete(i.traces, key)
		}
	}

	return nil
}

func (i *instance) IsBlockReady(maxTracesPerBlock int, maxBlockLifetime time.Duration) bool {
	i.blockTracesMtx.RLock()
	defer i.blockTracesMtx.RUnlock()

	now := time.Now()
	return len(i.traceRecords) >= maxTracesPerBlock || i.lastBlockCut.Add(maxBlockLifetime).Before(now)
}

// GetBlock() returns complete traces.  It is up to the caller to do something sensible at this point
func (i *instance) GetBlock() ([]*storage.TraceRecord, wal.WALBlock) {
	return i.traceRecords, i.walBlock
}

func (i *instance) ResetBlock() error {
	i.traceRecords = make([]*storage.TraceRecord, 0) //todo : init this with some value?  max traces per block?

	if i.walBlock != nil {
		i.walBlock.Clear()
	}

	var err error
	i.walBlock, err = i.wal.NewBlock(uuid.New(), i.instanceID)
	i.lastBlockCut = time.Now()
	return err
}

func (i *instance) FindTraceByID(id []byte) (*friggpb.Trace, error) {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	// search and return only complete traces.  traceRecords is ordered so binary search it
	idx := sort.Search(len(i.traceRecords), func(idx int) bool {
		return bytes.Compare(i.traceRecords[idx].TraceID, id) >= 0
	})

	if idx < 0 || idx >= len(i.traceRecords) {
		return nil, nil
	}

	rec := i.traceRecords[idx]
	if bytes.Compare(rec.TraceID, id) == 0 {
		i.blockTracesMtx.Lock()
		defer i.blockTracesMtx.Unlock()

		out := &friggpb.Trace{}

		err := i.walBlock.Read(rec.Start, rec.Length, out)
		if err != nil {
			return nil, err
		}

		return out, nil
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

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
