package ingester

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/pkg/model"
	v1 "github.com/grafana/tempo/pkg/model/v1"
	v2 "github.com/grafana/tempo/pkg/model/v2"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

var (
	ErrShuttingDown = errors.New("Ingester is shutting down")
	ErrStarting     = errors.New("Ingester is starting")
)

var metricFlushQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "tempo",
	Name:      "ingester_flush_queue_length",
	Help:      "The total number of series pending in the flush queue.",
})

const (
	ingesterRingKey = "ring"
)

// Ingester builds blocks out of incoming traces
type Ingester struct {
	services.Service

	cfg Config

	instancesMtx sync.RWMutex
	instances    map[string]*instance
	pushErr      atomic.Error

	lifecycler   *ring.Lifecycler
	store        storage.Store
	local        *local.Backend
	replayJitter bool // this var exists so tests can remove jitter

	flushQueues     *flushqueues.ExclusiveQueues
	flushQueuesDone sync.WaitGroup

	limiter *Limiter

	overrides ingesterOverrides

	subservicesWatcher *services.FailureWatcher
}

// New makes a new Ingester.
func New(cfg Config, store storage.Store, overrides overrides.Interface, reg prometheus.Registerer) (*Ingester, error) {
	i := &Ingester{
		cfg:          cfg,
		instances:    map[string]*instance{},
		store:        store,
		flushQueues:  flushqueues.New(cfg.ConcurrentFlushes, metricFlushQueueLength),
		replayJitter: true,
		overrides:    overrides,
	}

	i.pushErr.Store(ErrStarting)

	i.local = store.WAL().LocalBackend()

	lc, err := ring.NewLifecycler(cfg.LifecyclerConfig, i, "ingester", cfg.OverrideRingKey, true, log.Logger, prometheus.WrapRegistererWithPrefix("tempo_", reg))
	if err != nil {
		return nil, fmt.Errorf("NewLifecycler failed: %w", err)
	}
	i.lifecycler = lc

	// Now that the lifecycler has been created, we can create the limiter
	// which depends on it.
	i.limiter = NewLimiter(overrides, i.lifecycler, cfg.LifecyclerConfig.RingConfig.ReplicationFactor)

	i.subservicesWatcher = services.NewFailureWatcher()
	i.subservicesWatcher.WatchService(i.lifecycler)

	i.Service = services.NewBasicService(i.starting, i.loop, i.stopping)
	return i, nil
}

func (i *Ingester) starting(ctx context.Context) error {
	err := i.replayWal()
	if err != nil {
		return fmt.Errorf("failed to replay wal: %w", err)
	}

	err = i.rediscoverLocalBlocks()
	if err != nil {
		return fmt.Errorf("failed to rediscover local blocks: %w", err)
	}

	i.flushQueuesDone.Add(i.cfg.ConcurrentFlushes)
	for j := 0; j < i.cfg.ConcurrentFlushes; j++ {
		go i.flushLoop(j)
	}

	// Now that user states have been created, we can start the lifecycler.
	// Important: we want to keep lifecycler running until we ask it to stop, so we need to give it independent context
	if err := i.lifecycler.StartAsync(context.Background()); err != nil {
		return fmt.Errorf("failed to start lifecycler: %w", err)
	}
	if err := i.lifecycler.AwaitRunning(ctx); err != nil {
		return fmt.Errorf("failed to start lifecycle: %w", err)
	}

	i.pushErr.Store(nil)

	return nil
}

func (i *Ingester) loop(ctx context.Context) error {
	flushTicker := time.NewTicker(i.cfg.FlushCheckPeriod)
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			i.sweepAllInstances(false)

		case <-ctx.Done():
			return nil

		case err := <-i.subservicesWatcher.Chan():
			return fmt.Errorf("ingester subservice failed: %w", err)
		}
	}
}

// complete the flushing
// ExclusiveQueues.activekeys keeps track of flush operations due for processing
// ExclusiveQueues.IsEmpty check uses ExclusiveQueues.activeKeys to determine if flushQueues is empty or not
// sweepAllInstances prepares remaining traces to be flushed by flushLoop routine, also updating ExclusiveQueues.activekeys with keys for new flush operations
// ExclusiveQueues.activeKeys is cleared of a flush operation when a processing of flush operation is either successful or doesn't return retry signal
// This ensures that i.flushQueues is empty only when all traces are flushed
func (i *Ingester) flushRemaining() {
	i.sweepAllInstances(true)
	for !i.flushQueues.IsEmpty() {
		time.Sleep(100 * time.Millisecond)
	}
}

// stopping is run when ingester is asked to stop
func (i *Ingester) stopping(_ error) error {
	i.markUnavailable()

	// flush any remaining traces
	if i.cfg.FlushAllOnShutdown {
		i.flushRemaining()
	}

	if i.flushQueues != nil {
		i.flushQueues.Stop()
		i.flushQueuesDone.Wait()
	}

	i.local.Shutdown()

	return nil
}

func (i *Ingester) markUnavailable() {
	// Lifecycler can be nil if the ingester is for a flusher.
	if i.lifecycler != nil {
		// Next initiate our graceful exit from the ring.
		if err := services.StopAndAwaitTerminated(context.Background(), i.lifecycler); err != nil {
			level.Warn(log.Logger).Log("msg", "failed to stop ingester lifecycler", "err", err)
		}
	}

	// This will prevent us accepting any more samples
	i.stopIncomingRequests()
}

// PushBytes implements tempopb.Pusher.PushBytes. Traces pushed to this endpoint are expected to be in the formats
// defined by ./pkg/model/v1
// This push function is extremely inefficient and is only provided as a migration path from the v1->v2 encodings
func (i *Ingester) PushBytes(ctx context.Context, req *tempopb.PushBytesRequest) (*tempopb.PushResponse, error) {
	if err := i.pushErr.Load(); err != nil {
		return nil, err
	}

	var err error
	v1Decoder, err := model.NewSegmentDecoder(v1.Encoding)
	if err != nil {
		return nil, err
	}
	v2Decoder, err := model.NewSegmentDecoder(v2.Encoding)
	if err != nil {
		return nil, err
	}

	for i, t := range req.Traces {
		trace, err := v1Decoder.PrepareForRead([][]byte{t.Slice})
		if err != nil {
			return nil, fmt.Errorf("error calling v1.PrepareForRead: %w", err)
		}

		now := uint32(time.Now().Unix())
		v2Slice, err := v2Decoder.PrepareForWrite(trace, now, now)
		if err != nil {
			return nil, fmt.Errorf("error calling v2.PrepareForWrite: %w", err)
		}

		req.Traces[i].Slice = v2Slice
	}

	return i.PushBytesV2(ctx, req)
}

// PushBytes implements tempopb.Pusher.PushBytes. Traces pushed to this endpoint are expected to be in the formats
// defined by ./pkg/model/v2
func (i *Ingester) PushBytesV2(ctx context.Context, req *tempopb.PushBytesRequest) (*tempopb.PushResponse, error) {
	if err := i.pushErr.Load(); err != nil {
		return nil, err
	}

	if len(req.Traces) != len(req.Ids) {
		return nil, status.Errorf(codes.InvalidArgument, "mismatched traces/ids length: %d, %d", len(req.Traces), len(req.Ids))
	}

	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	instance, err := i.getOrCreateInstance(instanceID)
	if err != nil {
		level.Warn(log.Logger).Log(err.Error())
		return nil, err
	}

	return instance.PushBytesRequest(ctx, req), nil
}

// FindTraceByID implements tempopb.Querier.f
func (i *Ingester) FindTraceByID(ctx context.Context, req *tempopb.TraceByIDRequest) (res *tempopb.TraceByIDResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			level.Error(log.Logger).Log("msg", "recover in FindTraceByID", "id", util.TraceIDToHexString(req.TraceID), "stack", r, string(debug.Stack()))
			err = errors.New("recovered in FindTraceByID")
		}
	}()

	if !validation.ValidTraceID(req.TraceID) {
		return nil, fmt.Errorf("invalid trace id")
	}

	// tracing instrumentation
	span, ctx := opentracing.StartSpanFromContext(ctx, "Ingester.FindTraceByID")
	defer span.Finish()

	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.TraceByIDResponse{}, nil
	}

	trace, err := inst.FindTraceByID(ctx, req.TraceID)
	if err != nil {
		return nil, err
	}

	span.LogFields(ot_log.Bool("trace found", trace != nil))

	res = &tempopb.TraceByIDResponse{
		Trace: trace,
	}

	return res, nil
}

func (i *Ingester) CheckReady(ctx context.Context) error {
	if err := i.lifecycler.CheckReady(ctx); err != nil {
		return fmt.Errorf("ingester check ready failed: %w", err)
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
		inst, err = newInstance(instanceID, i.limiter, i.overrides, i.store, i.local, i.cfg.AutocompleteFilteringEnabled, i.cfg.DedicatedColumns)
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

	i.pushErr.Store(ErrShuttingDown)
}

// TransferOut implements ring.Lifecycler.
func (i *Ingester) TransferOut(context.Context) error {
	return ring.ErrTransferDisabled
}

func (i *Ingester) replayWal() error {
	level.Info(log.Logger).Log("msg", "beginning wal replay")

	// pass i.cfg.MaxBlockDuration into RescanBlocks to make an attempt to set the start time
	// of the blocks correctly. as we are scanning traces in the blocks we read their start/end times
	// and attempt to set start/end times appropriately. we use now - max_block_duration - ingestion_slack
	// as the minimum acceptable start time for a replayed block.
	blocks, err := i.store.WAL().RescanBlocks(i.cfg.MaxBlockDuration, log.Logger)
	if err != nil {
		return fmt.Errorf("fatal error replaying wal: %w", err)
	}

	for _, b := range blocks {
		tenantID := b.BlockMeta().TenantID

		instance, err := i.getOrCreateInstance(tenantID)
		if err != nil {
			return err
		}

		// Delete anything remaining for the completed version of this
		// wal block. This handles the case where a wal file is partially
		// or fully completed to the local store, but the wal file wasn't
		// deleted (because it was rescanned above). This can happen for reasons
		// such as a crash or restart. In this situation we err on the side of
		// caution and replay the wal block.
		err = instance.local.ClearBlock(b.BlockMeta().BlockID, tenantID)
		if err != nil {
			return err
		}
		instance.AddCompletingBlock(b)

		i.enqueue(&flushOp{
			kind:    opKindComplete,
			userID:  tenantID,
			blockID: b.BlockMeta().BlockID,
		}, i.replayJitter)
	}

	level.Info(log.Logger).Log("msg", "wal replay complete")

	return nil
}

func (i *Ingester) rediscoverLocalBlocks() error {
	ctx := context.TODO()

	reader := backend.NewReader(i.local)
	tenants, err := reader.Tenants(ctx)
	if err != nil {
		return fmt.Errorf("getting local tenants: %w", err)
	}

	level.Info(log.Logger).Log("msg", "reloading local blocks", "tenants", len(tenants))

	for _, t := range tenants {
		// check if any local blocks exist for a tenant before creating the instance. this is to protect us from cases
		// where left-over empty local tenant folders persist empty tenants
		blocks, _, err := reader.Blocks(ctx, t)
		if err != nil {
			return err
		}
		if len(blocks) == 0 {
			continue
		}

		inst, err := i.getOrCreateInstance(t)
		if err != nil {
			return err
		}

		newBlocks, err := inst.rediscoverLocalBlocks(ctx)
		if err != nil {
			return fmt.Errorf("getting local blocks for tenant %v: %w", t, err)
		}

		// Requeue needed flushes
		for _, b := range newBlocks {
			if b.FlushedTime().IsZero() {
				i.enqueue(&flushOp{
					kind:    opKindFlush,
					userID:  t,
					blockID: b.BlockMeta().BlockID,
				}, i.replayJitter)
			}
		}
	}

	return nil
}
