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
	"github.com/google/uuid"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
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

var tracer = otel.Tracer("modules/ingester")

const (
	ingesterRingKey = "ring"

	// PartitionRingKey is the key under which we store the partitions ring used by the "ingest storage".
	PartitionRingKey  = "ingester-partitions"
	PartitionRingName = "ingester-partitions"
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

	// manages synchronous behavior with startCutToWal
	cutToWalWg    sync.WaitGroup
	cutToWalStop  chan struct{}
	cutToWalStart chan struct{}
	limiter       Limiter

	// Used by ingest storage when enabled
	ingestPartitionLifecycler *ring.PartitionInstanceLifecycler
	ingestPartitionID         int32

	overrides ingesterOverrides

	subservicesWatcher *services.FailureWatcher
}

// New makes a new Ingester.
func New(cfg Config, store storage.Store, overrides overrides.Interface, reg prometheus.Registerer, singlePartition bool) (*Ingester, error) {
	i := &Ingester{
		cfg:          cfg,
		instances:    map[string]*instance{},
		store:        store,
		flushQueues:  flushqueues.New(cfg.ConcurrentFlushes, metricFlushQueueLength),
		replayJitter: true,
		overrides:    overrides,

		cutToWalStart: make(chan struct{}),
		cutToWalStop:  make(chan struct{}),
	}

	i.pushErr.Store(ErrStarting)

	i.local = store.WAL().LocalBackend()

	lc, err := ring.NewLifecycler(cfg.LifecyclerConfig, nil, "ingester", cfg.OverrideRingKey, true, log.Logger, prometheus.WrapRegistererWithPrefix("tempo_", reg))
	if err != nil {
		return nil, fmt.Errorf("NewLifecycler failed: %w", err)
	}
	i.lifecycler = lc

	if cfg.IngestStorageConfig.Enabled {
		if singlePartition {
			// For single-binary don't require hostname to identify a partition.
			// Assume partition 0.
			i.ingestPartitionID = 0
		} else {
			i.ingestPartitionID, err = ingest.IngesterPartitionID(cfg.LifecyclerConfig.ID)
			if err != nil {
				return nil, fmt.Errorf("calculating ingester partition ID: %w", err)
			}
		}

		partitionRingKV := cfg.IngesterPartitionRing.KVStore.Mock
		if partitionRingKV == nil {
			partitionRingKV, err = kv.NewClient(cfg.IngesterPartitionRing.KVStore, ring.GetPartitionRingCodec(), kv.RegistererWithKVName(reg, PartitionRingName+"-lifecycler"), log.Logger)
			if err != nil {
				return nil, fmt.Errorf("creating KV store for ingester partition ring: %w", err)
			}
		}

		i.ingestPartitionLifecycler = ring.NewPartitionInstanceLifecycler(
			i.cfg.IngesterPartitionRing.ToLifecyclerConfig(i.ingestPartitionID, cfg.LifecyclerConfig.ID),
			PartitionRingName,
			PartitionRingKey,
			partitionRingKV,
			log.Logger,
			prometheus.WrapRegistererWithPrefix("cortex_", reg))
	}

	// Now that the lifecycler has been created, we can create the limiter
	// which depends on it.
	i.limiter = NewLimiter(overrides, i.lifecycler, cfg.LifecyclerConfig.RingConfig.ReplicationFactor)

	i.subservicesWatcher = services.NewFailureWatcher()
	i.subservicesWatcher.WatchService(i.lifecycler)
	if cfg.IngestStorageConfig.Enabled {
		i.subservicesWatcher.WatchService(i.ingestPartitionLifecycler)
	}

	i.Service = services.NewBasicService(i.starting, i.running, i.stopping)
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

	if i.ingestPartitionLifecycler != nil {
		if err := i.ingestPartitionLifecycler.StartAsync(context.Background()); err != nil {
			return fmt.Errorf("failed to start ingest partition lifecycler: %w", err)
		}
		if err := i.ingestPartitionLifecycler.AwaitRunning(ctx); err != nil {
			return fmt.Errorf("failed to start ingest partition lifecycle: %w", err)
		}
	}

	// accept traces
	i.pushErr.Store(nil)

	// start flushing traces to wal
	close(i.cutToWalStart)

	return nil
}

func (i *Ingester) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-i.subservicesWatcher.Chan():
		return fmt.Errorf("ingester subservice failed: %w", err)
	}
}

// stopping is run when ingester is asked to stop
func (i *Ingester) stopping(_ error) error {
	i.markUnavailable()

	// signal all cutting to wal to stop and wait for all goroutines to finish
	close(i.cutToWalStop)
	i.cutToWalWg.Wait()

	if i.cfg.FlushAllOnShutdown {
		// force all in memory traces to be flushed to disk AND fully flush them to the backend
		i.flushRemaining()
	} else {
		// force all in memory traces to be flushed to disk
		i.cutAllInstancesToWal()
	}

	if i.flushQueues != nil {
		i.flushQueues.Stop()
		i.flushQueuesDone.Wait()
	}

	i.local.Shutdown()

	return nil
}

// complete the flushing
// ExclusiveQueues.activekeys keeps track of flush operations due for processing
// ExclusiveQueues.IsEmpty check uses ExclusiveQueues.activeKeys to determine if flushQueues is empty or not
// sweepAllInstances prepares remaining traces to be flushed by flushLoop routine, also updating ExclusiveQueues.activekeys with keys for new flush operations
// ExclusiveQueues.activeKeys is cleared of a flush operation when a processing of flush operation is either successful or doesn't return retry signal
// This ensures that i.flushQueues is empty only when all traces are flushed
func (i *Ingester) flushRemaining() {
	i.cutAllInstancesToWal()
	for !i.flushQueues.IsEmpty() {
		time.Sleep(100 * time.Millisecond)
	}
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
	i.pushErr.Store(ErrShuttingDown)
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
	ctx, span := tracer.Start(ctx, "Ingester.FindTraceByID")
	defer span.End()

	instanceID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}
	inst, ok := i.getInstanceByID(instanceID)
	if !ok || inst == nil {
		return &tempopb.TraceByIDResponse{}, nil
	}

	res, err = inst.FindTraceByID(ctx, req.TraceID, req.AllowPartialTrace)
	if err != nil {
		return nil, err
	}

	span.AddEvent("trace found", oteltrace.WithAttributes(attribute.Bool("found", res != nil && res.Trace != nil)))

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
		inst, err = newInstance(instanceID, i.limiter, i.overrides, i.store, i.local, i.cfg.DedicatedColumns)
		if err != nil {
			return nil, err
		}
		i.instances[instanceID] = inst

		i.cutToWalLoop(inst)
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
		err = instance.local.ClearBlock((uuid.UUID)(b.BlockMeta().BlockID), tenantID)
		if err != nil {
			return err
		}
		instance.AddCompletingBlock(b)

		i.enqueue(&flushOp{
			kind:    opKindComplete,
			userID:  tenantID,
			blockID: (uuid.UUID)(b.BlockMeta().BlockID),
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
		if i.cfg.FlushObjectStorage {
			for _, b := range newBlocks {
				if b.FlushedTime().IsZero() {
					i.enqueue(&flushOp{
						kind:    opKindFlush,
						userID:  t,
						blockID: (uuid.UUID)(b.BlockMeta().BlockID),
					}, i.replayJitter)
				}
			}
		}
	}

	return nil
}
