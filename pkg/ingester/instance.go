package ingester

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"

	"github.com/joe-elliott/frigg/pkg/friggpb"
	"github.com/joe-elliott/frigg/pkg/util"
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
	traces    map[traceFingerprint]*trace // we use 'mapped' fingerprints here.

	blockTracesMtx sync.RWMutex
	blockTraces    []*trace // friggtodo: init this with some configurable large value?
	lastBlockCut   time.Time

	instanceID         string
	tracesCreatedTotal prometheus.Counter
	limiter            *Limiter
}

func newInstance(instanceID string, limiter *Limiter) *instance {
	i := &instance{
		traces:       map[traceFingerprint]*trace{},
		lastBlockCut: time.Now(),

		instanceID:         instanceID,
		tracesCreatedTotal: tracesCreatedTotal.WithLabelValues(instanceID),
		limiter:            limiter,
	}
	return i
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
func (i *instance) CutCompleteTraces(cutoff time.Duration, immediate bool) {
	i.tracesMtx.Lock()
	defer i.tracesMtx.Unlock()

	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	now := time.Now()
	for key, trace := range i.traces {
		if now.Add(cutoff).After(trace.lastAppend) || immediate {
			i.blockTraces = append(i.blockTraces, trace)
			delete(i.traces, key)
		}
	}
}

func (i *instance) IsBlockReady(maxTracesPerBlock int, maxBlockLifetime time.Duration) bool {
	i.blockTracesMtx.RLock()
	defer i.blockTracesMtx.RUnlock()

	now := time.Now()
	return len(i.blockTraces) >= maxTracesPerBlock && i.lastBlockCut.Add(maxBlockLifetime).After(now)
}

// GetBlock() returns complete traces.  It is up to the caller to do something sensible at this point
func (i *instance) GetBlock() []*trace {
	i.blockTracesMtx.Lock()
	defer i.blockTracesMtx.Unlock()

	ret := i.blockTraces
	i.blockTraces = []*trace{}

	return ret
}

func (i *instance) getOrCreateTrace(req *friggpb.PushRequest) (*trace, error) {
	fp := traceFingerprint(util.Fingerprint(req.Spans[0].TraceID)) // friggtodo:  drop this assumption?

	trace, ok := i.traces[fp]
	if ok {
		return trace, nil
	}

	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, len(i.traces))
	if err != nil {
		return nil, httpgrpc.Errorf(http.StatusTooManyRequests, err.Error())
	}

	trace = newTrace(fp)
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
