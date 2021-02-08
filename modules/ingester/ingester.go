package ingester

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/services"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/validation"
	tempodb_wal "github.com/grafana/tempo/tempodb/wal"
)

// ErrReadOnly is returned when the ingester is shutting down and a push was
// attempted.
var ErrReadOnly = errors.New("Ingester is shutting down")

var metricFlushQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "tempo",
	Name:      "ingester_flush_queue_length",
	Help:      "The total number of series pending in the flush queue.",
})

// Ingester builds blocks out of incoming traces
type Ingester struct {
	services.Service

	cfg Config

	instancesMtx sync.RWMutex
	instances    map[string]*instance
	readonly     bool

	lifecycler *ring.Lifecycler
	store      storage.Store

	flushQueues     *flushqueues.ExclusiveQueues
	flushQueuesDone sync.WaitGroup

	limiter *Limiter

	subservicesWatcher *services.FailureWatcher
}

// New makes a new Ingester.
func New(cfg Config, store storage.Store, limits *overrides.Overrides) (*Ingester, error) {
	i := &Ingester{
		cfg:         cfg,
		instances:   map[string]*instance{},
		store:       store,
		flushQueues: flushqueues.New(cfg.ConcurrentFlushes, metricFlushQueueLength),
	}

	i.flushQueuesDone.Add(cfg.ConcurrentFlushes)
	for j := 0; j < cfg.ConcurrentFlushes; j++ {
		go i.flushLoop(j)
	}

	var err error
	i.lifecycler, err = ring.NewLifecycler(cfg.LifecyclerConfig, i, "ingester", cfg.OverrideRingKey, true, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, fmt.Errorf("NewLifecycler failed %w", err)
	}

	// Now that the lifecycler has been created, we can create the limiter
	// which depends on it.
	i.limiter = NewLimiter(limits, i.lifecycler, cfg.LifecyclerConfig.RingConfig.ReplicationFactor)

	i.subservicesWatcher = services.NewFailureWatcher()
	i.subservicesWatcher.WatchService(i.lifecycler)

	i.Service = services.NewBasicService(i.starting, i.loop, i.stopping)
	return i, nil
}

func (i *Ingester) starting(ctx context.Context) error {
	// Now that user states have been created, we can start the lifecycler.
	// Important: we want to keep lifecycler running until we ask it to stop, so we need to give it independent context
	if err := i.lifecycler.StartAsync(context.Background()); err != nil {
		return fmt.Errorf("failed to start lifecycler %w", err)
	}
	if err := i.lifecycler.AwaitRunning(ctx); err != nil {
		return fmt.Errorf("failed to start lifecycle %w", err)
	}

	err := i.replayWal()
	if err != nil {
		return fmt.Errorf("failed to replay wal %w", err)
	}

	return nil
}

func (i *Ingester) loop(ctx context.Context) error {
	flushTicker := time.NewTicker(i.cfg.FlushCheckPeriod)
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			i.sweepUsers(false)

		case <-ctx.Done():
			return nil

		case err := <-i.subservicesWatcher.Chan():
			return fmt.Errorf("ingester subservice failed %w", err)
		}
	}
}

// stopping is run when ingester is asked to stop
func (i *Ingester) stopping(_ error) error {
	// This will prevent us accepting any more samples
	i.stopIncomingRequests()

	// Lifecycler can be nil if the ingester is for a flusher.
	if i.lifecycler != nil {
		// Next initiate our graceful exit from the ring.
		return services.StopAndAwaitTerminated(context.Background(), i.lifecycler)
	}

	if i.flushQueues != nil {
		i.flushQueues.Stop()
		i.flushQueuesDone.Wait()
	}

	return nil
}

// Push implements tempopb.Pusher.Push
func (i *Ingester) Push(ctx context.Context, req *tempopb.PushRequest) (*tempopb.PushResponse, error) {
	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	} else if i.readonly {
		return nil, ErrReadOnly
	}

	instance, err := i.getOrCreateInstance(instanceID)
	if err != nil {
		return nil, err
	}

	err = instance.Push(ctx, req)
	return &tempopb.PushResponse{}, err
}

// PushBytes implements tempopb.Pusher.PushBytes
func (i *Ingester) PushBytes(ctx context.Context, req *tempopb.PushBytesRequest) (*tempopb.PushResponse, error) {

	// Unmarshal and push each request
	for _, v := range req.Requests {
		r := tempopb.PushRequest{}
		err := r.Unmarshal(v)
		if err != nil {
			return nil, err
		}

		_, err = i.Push(ctx, &r)
		if err != nil {
			return nil, err
		}
	}
	return &tempopb.PushResponse{}, nil
}

// FindTraceByID implements tempopb.Querier.f
func (i *Ingester) FindTraceByID(ctx context.Context, req *tempopb.TraceByIDRequest) (*tempopb.TraceByIDResponse, error) {
	if !validation.ValidTraceID(req.TraceID) {
		return nil, fmt.Errorf("invalid trace id")
	}

	// tracing instrumentation
	span, ctx := opentracing.StartSpanFromContext(ctx, "ingester.FindTraceByID")
	defer span.Finish()

	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.TraceByIDResponse{}, nil
	}

	trace, err := inst.FindTraceByID(req.TraceID)
	if err != nil {
		return nil, err
	}

	return &tempopb.TraceByIDResponse{
		Trace: trace,
	}, nil
}

func (i *Ingester) CheckReady(ctx context.Context) error {
	if err := i.lifecycler.CheckReady(ctx); err != nil {
		return fmt.Errorf("ingester check ready failed %w", err)
	}

	return nil
}

func (i *Ingester) getOrCreateInstance(instanceID string) (*instance, error) {
	inst, ok := i.getInstanceByID(instanceID)
	if ok {
		return inst, nil
	}

	i.instancesMtx.Lock()
	defer i.instancesMtx.Unlock()
	inst, ok = i.instances[instanceID]
	if !ok {
		var err error
		inst, err = newInstance(instanceID, i.limiter, i.store)
		if err != nil {
			return nil, err
		}
		i.instances[instanceID] = inst
	}
	return inst, nil
}

func (i *Ingester) getInstanceByID(id string) (*instance, bool) {
	i.instancesMtx.RLock()
	defer i.instancesMtx.RUnlock()

	inst, ok := i.instances[id]
	return inst, ok
}

func (i *Ingester) getInstances() []*instance {
	i.instancesMtx.RLock()
	defer i.instancesMtx.RUnlock()

	instances := make([]*instance, 0, len(i.instances))
	for _, instance := range i.instances {
		instances = append(instances, instance)
	}
	return instances
}

// stopIncomingRequests implements ring.Lifecycler.
func (i *Ingester) stopIncomingRequests() {
	i.instancesMtx.Lock()
	defer i.instancesMtx.Unlock()

	i.readonly = true
}

// TransferOut implements ring.Lifecycler.
func (i *Ingester) TransferOut(ctx context.Context) error {
	return ring.ErrTransferDisabled
}

func (i *Ingester) replayWal() error {
	blocks, err := i.store.WAL().AllBlocks()
	// todo: should this fail startup?
	if err != nil {
		return nil
	}

	level.Info(util.Logger).Log("msg", "beginning wal replay", "numBlocks", len(blocks))

	for _, b := range blocks {
		tenantID := b.TenantID()
		level.Info(util.Logger).Log("msg", "beginning block replay", "tenantID", tenantID)

		instance, err := i.getOrCreateInstance(tenantID)
		if err != nil {
			return err
		}

		err = i.replayBlock(b, instance)
		if err != nil {
			// there was an error, log and keep on keeping on
			level.Error(util.Logger).Log("msg", "error replaying block.  removing", "error", err)
		}
		err = b.Clear()
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Ingester) replayBlock(b *tempodb_wal.ReplayBlock, instance *instance) error {
	iterator, err := b.Iterator()
	if err != nil {
		return err
	}
	for {
		id, obj, err := iterator.Next()
		if id == nil {
			break
		}
		if err != nil {
			return err
		}

		// obj gets written to disk immediately but the id escapes the iterator and needs to be copied
		writeID := append([]byte(nil), id...)
		err = instance.PushBytes(context.Background(), writeID, obj)
		if err != nil {
			return err
		}
	}

	return nil
}
