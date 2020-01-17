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
	memoryTraces = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "frigg",
		Name:      "ingester_memory_traces",
		Help:      "The total number of traces in memory.",
	})
	tracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "ingester_traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	tracesRemovedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "frigg",
		Name:      "ingester_traces_removed_total",
		Help:      "The total number of traces removed per tenant.",
	}, []string{"tenant"})
)

type instance struct {
	tracesMtx sync.RWMutex
	traces    map[traceFingerprint]*trace // we use 'mapped' fingerprints here.

	instanceID string

	tracesCreatedTotal prometheus.Counter
	tracesRemovedTotal prometheus.Counter

	limiter *Limiter

	// sync
	syncPeriod  time.Duration
	syncMinUtil float64
}

func newInstance(instanceID string, factory func() chunkenc.Chunk, limiter *Limiter, syncPeriod time.Duration, syncMinUtil float64) *instance {
	i := &instance{
		traces:     map[traceFingerprint]*trace{},
		instanceID: instanceID,

		tracesCreatedTotal: tracesCreatedTotal.WithLabelValues(instanceID),
		tracesRemovedTotal: tracesRemovedTotal.WithLabelValues(instanceID),

		factory: factory,
		limiter: limiter,

		syncPeriod:  syncPeriod,
		syncMinUtil: syncMinUtil,
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

	if err := trace.Push(ctx, req, i.syncPeriod, i.syncMinUtil); err != nil {
		return err
	}
}

func (i *instance) getOrCreateTrace(req *friggpb.PushRequest) (*trace, error) {
	fp := util.Fingerprint(req.Spans[0].TraceID) // friggtodo:  drop this assumption?

	trace, ok := i.traces[fp]
	if ok {
		return trace, nil
	}

	err := i.limiter.AssertMaxTracesPerUser(i.instanceID, len(i.traces))
	if err != nil {
		return nil, httpgrpc.Errorf(http.StatusTooManyRequests, err.Error())
	}

	trace = newTrace(fp)
	i.traces[fp] = stream
	memoryTraces.Inc()
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
