package compactor

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
)

const (
	// ringAutoForgetUnhealthyPeriods is how many consecutive timeout periods an unhealthy instance
	// in the ring will be automatically removed.
	ringAutoForgetUnhealthyPeriods = 2

	// We use a safe default instead of exposing to config option to the user
	// in order to simplify the config.
	ringNumTokens = 512

	compactorRingKey = "compactor"

	reasonCompactorDiscardedSpans = "trace_too_large_to_compact"
)

var ringOp = ring.NewOp([]ring.InstanceState{ring.ACTIVE}, nil)

type Compactor struct {
	services.Service

	cfg       *Config
	store     storage.Store
	overrides overrides.Interface

	// Ring used for sharding compactions.
	ringLifecycler *ring.BasicLifecycler
	Ring           *ring.Ring

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New makes a new Compactor.
func New(cfg Config, store storage.Store, overrides overrides.Interface, reg prometheus.Registerer) (*Compactor, error) {
	c := &Compactor{
		cfg:       &cfg,
		store:     store,
		overrides: overrides,
	}

	if c.isSharded() {
		reg = prometheus.WrapRegistererWithPrefix("tempo_", reg)

		lifecyclerStore, err := kv.NewClient(
			cfg.ShardingRing.KVStore,
			ring.GetCodec(),
			kv.RegistererWithKVName(reg, compactorRingKey+"-lifecycler"),
			log.Logger,
		)
		if err != nil {
			return nil, err
		}

		delegate := ring.BasicLifecyclerDelegate(c)
		delegate = ring.NewLeaveOnStoppingDelegate(delegate, log.Logger)
		delegate = ring.NewAutoForgetDelegate(ringAutoForgetUnhealthyPeriods*cfg.ShardingRing.HeartbeatTimeout, delegate, log.Logger)

		bcfg, err := toBasicLifecyclerConfig(cfg.ShardingRing, log.Logger)
		if err != nil {
			return nil, err
		}

		c.ringLifecycler, err = ring.NewBasicLifecycler(bcfg, compactorRingKey, cfg.OverrideRingKey, lifecyclerStore, delegate, log.Logger, reg)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize compactor ring lifecycler: %w", err)
		}

		c.Ring, err = ring.New(c.cfg.ShardingRing.ToLifecyclerConfig().RingConfig, compactorRingKey, cfg.OverrideRingKey, log.Logger, reg)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize compactor ring: %w", err)
		}
	}

	c.Service = services.NewBasicService(c.starting, c.running, c.stopping)

	return c, nil
}

func (c *Compactor) starting(ctx context.Context) (err error) {
	// In case this function will return error we want to unregister the instance
	// from the ring. We do it ensuring dependencies are gracefully stopped if they
	// were already started.
	defer func() {
		if err == nil || c.subservices == nil {
			return
		}

		if stopErr := services.StopManagerAndAwaitStopped(context.Background(), c.subservices); stopErr != nil {
			level.Error(log.Logger).Log("msg", "failed to gracefully stop compactor dependencies", "err", stopErr)
		}
	}()

	if c.isSharded() {
		c.subservices, err = services.NewManager(c.ringLifecycler, c.Ring)
		if err != nil {
			return fmt.Errorf("failed to create subservices: %w", err)
		}
		c.subservicesWatcher = services.NewFailureWatcher()
		c.subservicesWatcher.WatchManager(c.subservices)

		err := services.StartManagerAndAwaitHealthy(ctx, c.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices: %w", err)
		}

		// Wait until the ring client detected this instance in the ACTIVE state.
		level.Info(log.Logger).Log("msg", "waiting until compactor is ACTIVE in the ring")
		ctxWithTimeout, cancel := context.WithTimeout(ctx, c.cfg.ShardingRing.WaitActiveInstanceTimeout)
		defer cancel()
		if err := ring.WaitInstanceState(ctxWithTimeout, c.Ring, c.ringLifecycler.GetInstanceID(), ring.ACTIVE); err != nil {
			return err
		}
		level.Info(log.Logger).Log("msg", "compactor is ACTIVE in the ring")

		// In the event of a cluster cold start we may end up in a situation where each new compactor
		// instance starts at a slightly different time and thus each one starts with a different state
		// of the ring. It's better to just wait the ring stability for a short time.
		if c.cfg.ShardingRing.WaitStabilityMinDuration > 0 {
			minWaiting := c.cfg.ShardingRing.WaitStabilityMinDuration
			maxWaiting := c.cfg.ShardingRing.WaitStabilityMaxDuration

			level.Info(log.Logger).Log("msg", "waiting until compactor ring topology is stable", "min_waiting", minWaiting.String(), "max_waiting", maxWaiting.String())
			if err := ring.WaitRingStability(ctx, c.Ring, ringOp, minWaiting, maxWaiting); err != nil {
				level.Warn(log.Logger).Log("msg", "compactor ring topology is not stable after the max waiting time, proceeding anyway")
			} else {
				level.Info(log.Logger).Log("msg", "compactor ring topology is stable")
			}
		}
	}

	// this will block until one poll cycle is complete
	c.store.EnablePolling(ctx, c)

	return nil
}

func (c *Compactor) running(ctx context.Context) error {
	if !c.cfg.Disabled {
		level.Info(log.Logger).Log("msg", "enabling compaction")
		err := c.store.EnableCompaction(ctx, &c.cfg.Compactor, c, c)
		if err != nil {
			return fmt.Errorf("failed to enable compaction: %w", err)
		}
	}

	if c.subservices != nil {
		select {
		case <-ctx.Done():
			return nil
		case err := <-c.subservicesWatcher.Chan():
			return fmt.Errorf("compactor subservices failed: %w", err)
		}
	} else {
		<-ctx.Done()
	}

	return nil
}

// Called after compactor is asked to stop via StopAsync.
func (c *Compactor) stopping(_ error) error {
	if c.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), c.subservices)
	}

	return nil
}

// Owns implements tempodb.CompactorSharder
func (c *Compactor) Owns(hash string) bool {
	if !c.isSharded() {
		return true
	}

	level.Debug(log.Logger).Log("msg", "checking hash", "hash", hash)

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(hash))
	hash32 := hasher.Sum32()

	rs, err := c.Ring.Get(hash32, ringOp, []ring.InstanceDesc{}, nil, nil)
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get ring", "err", err)
		return false
	}

	if len(rs.Instances) != 1 {
		level.Error(log.Logger).Log("msg", "unexpected number of compactors in the shard (expected 1, got %d)", len(rs.Instances))
		return false
	}

	ringAddr := c.ringLifecycler.GetInstanceAddr()

	level.Debug(log.Logger).Log("msg", "checking addresses", "owning_addr", rs.Instances[0].Addr, "this_addr", ringAddr)

	return rs.Instances[0].Addr == ringAddr
}

// Combine implements tempodb.CompactorSharder
func (c *Compactor) Combine(dataEncoding string, tenantID string, objs ...[]byte) ([]byte, bool, error) {
	combinedObj, wasCombined, err := model.StaticCombiner.Combine(dataEncoding, objs...)
	if err != nil {
		return nil, false, err
	}

	maxBytes := c.overrides.MaxBytesPerTrace(tenantID)
	if maxBytes == 0 || len(combinedObj) < maxBytes {
		return combinedObj, wasCombined, nil
	}

	// technically neither of these conditions should ever be true, we are adding them as guard code
	// for the following logic
	if len(objs) == 0 {
		return []byte{}, wasCombined, nil
	}
	if len(objs) == 1 {
		return objs[0], wasCombined, nil
	}

	totalDiscarded := countSpans(dataEncoding, objs[1:]...)
	overrides.RecordDiscardedSpans(totalDiscarded, reasonCompactorDiscardedSpans, tenantID)
	return objs[0], wasCombined, nil
}

// RecordDiscardedSpans implements tempodb.CompactorSharder
func (c *Compactor) RecordDiscardedSpans(count int, tenantID string, traceID string, rootSpanName string, rootServiceName string) {
	level.Warn(log.Logger).Log("msg", "max size of trace exceeded", "tenant", tenantID, "traceId", traceID,
		"rootSpanName", rootSpanName, "rootServiceName", rootServiceName, "discarded_span_count", count)
	overrides.RecordDiscardedSpans(count, reasonCompactorDiscardedSpans, tenantID)
}

// BlockRetentionForTenant implements CompactorOverrides
func (c *Compactor) BlockRetentionForTenant(tenantID string) time.Duration {
	return c.overrides.BlockRetention(tenantID)
}

func (c *Compactor) MaxBytesPerTraceForTenant(tenantID string) int {
	return c.overrides.MaxBytesPerTrace(tenantID)
}

func (c *Compactor) isSharded() bool {
	return c.cfg.ShardingRing.KVStore.Store != ""
}

// OnRingInstanceRegister is called while the lifecycler is registering the
// instance within the ring and should return the state and set of tokens to
// use for the instance itself.
func (c *Compactor) OnRingInstanceRegister(_ *ring.BasicLifecycler, ringDesc ring.Desc, instanceExists bool, _ string, instanceDesc ring.InstanceDesc) (ring.InstanceState, ring.Tokens) {
	// When we initialize the compactor instance in the ring we want to start from
	// a clean situation, so whatever is the state we set it ACTIVE, while we keep existing
	// tokens (if any) or the ones loaded from file.
	var tokens []uint32
	if instanceExists {
		tokens = instanceDesc.GetTokens()
	}

	takenTokens := ringDesc.GetTokens()
	gen := ring.NewRandomTokenGenerator()
	newTokens := gen.GenerateTokens(ringNumTokens-len(tokens), takenTokens)

	// Tokens sorting will be enforced by the parent caller.
	tokens = append(tokens, newTokens...)

	return ring.ACTIVE, tokens
}

// OnRingInstanceTokens is called once the instance tokens are set and are
// stable within the ring (honoring the observe period, if set).
func (c *Compactor) OnRingInstanceTokens(*ring.BasicLifecycler, ring.Tokens) {}

// OnRingInstanceStopping is called while the lifecycler is stopping. The lifecycler
// will continue to hearbeat the ring the this function is executing and will proceed
// to unregister the instance from the ring only after this function has returned.
func (c *Compactor) OnRingInstanceStopping(*ring.BasicLifecycler) {}

// OnRingInstanceHeartbeat is called while the instance is updating its heartbeat
// in the ring.
func (c *Compactor) OnRingInstanceHeartbeat(*ring.BasicLifecycler, *ring.Desc, *ring.InstanceDesc) {
}

func countSpans(dataEncoding string, objs ...[]byte) (total int) {
	var traceID string
	decoder, err := model.NewObjectDecoder(dataEncoding)
	if err != nil {
		return 0
	}

	for _, o := range objs {
		t, err := decoder.PrepareForRead(o)
		if err != nil {
			continue
		}

		for _, b := range t.Batches {
			for _, ilm := range b.ScopeSpans {
				if len(ilm.Spans) > 0 && traceID == "" {
					traceID = tempoUtil.TraceIDToHexString(ilm.Spans[0].TraceId)
				}
				total += len(ilm.Spans)
			}
		}
	}

	level.Debug(log.Logger).Log("msg", "max size of trace exceeded", "traceId", traceID, "discarded_span_count", total)

	return
}
