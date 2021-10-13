package compactor

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/model"
)

const (
	waitOnStartup = time.Minute
)

type Compactor struct {
	services.Service

	cfg       *Config
	store     storage.Store
	overrides *overrides.Overrides

	// Ring used for sharding compactions.
	ringLifecycler *ring.Lifecycler
	Ring           *ring.Ring

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New makes a new Compactor.
func New(cfg Config, store storage.Store, overrides *overrides.Overrides) (*Compactor, error) {
	c := &Compactor{
		cfg:       &cfg,
		store:     store,
		overrides: overrides,
	}

	subservices := []services.Service(nil)
	if c.isSharded() {
		lifecyclerCfg := c.cfg.ShardingRing.ToLifecyclerConfig()
		lifecycler, err := ring.NewLifecycler(lifecyclerCfg, ring.NewNoopFlushTransferer(), "compactor", cfg.OverrideRingKey, false, log.Logger, prometheus.DefaultRegisterer)
		if err != nil {
			return nil, errors.Wrap(err, "unable to initialize compactor ring lifecycler")
		}
		c.ringLifecycler = lifecycler
		subservices = append(subservices, c.ringLifecycler)

		ring, err := ring.New(lifecyclerCfg.RingConfig, "compactor", cfg.OverrideRingKey, log.Logger, prometheus.DefaultRegisterer)
		if err != nil {
			return nil, errors.Wrap(err, "unable to initialize compactor ring")
		}
		c.Ring = ring
		subservices = append(subservices, c.Ring)

		c.subservices, err = services.NewManager(subservices...)
		if err != nil {
			return nil, fmt.Errorf("failed to create subservices %w", err)
		}
		c.subservicesWatcher = services.NewFailureWatcher()
		c.subservicesWatcher.WatchManager(c.subservices)
	}

	c.Service = services.NewBasicService(c.starting, c.running, c.stopping)

	return c, nil
}

func (c *Compactor) starting(ctx context.Context) error {
	if c.subservices != nil {
		err := services.StartManagerAndAwaitHealthy(ctx, c.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices %w", err)
		}

		ctx := context.Background()

		level.Info(log.Logger).Log("msg", "waiting to be active in the ring")
		err = c.waitRingActive(ctx)
		if err != nil {
			return err
		}
	}

	// this will block until one poll cycle is complete
	c.store.EnablePolling(c)

	return nil
}

func (c *Compactor) running(ctx context.Context) error {
	go func() {
		level.Info(log.Logger).Log("msg", "waiting for compaction ring to settle", "waitDuration", waitOnStartup)
		time.Sleep(waitOnStartup)
		level.Info(log.Logger).Log("msg", "enabling compaction")
		c.store.EnableCompaction(&c.cfg.Compactor, c, c)
	}()

	if c.subservices != nil {
		select {
		case <-ctx.Done():
			return nil
		case err := <-c.subservicesWatcher.Chan():
			return fmt.Errorf("distributor subservices failed %w", err)
		}
	} else {
		<-ctx.Done()
	}

	return nil
}

// Called after distributor is asked to stop via StopAsync.
func (c *Compactor) stopping(_ error) error {
	if c.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), c.subservices)
	}

	return nil
}

// Owns implements CompactorSharder
func (c *Compactor) Owns(hash string) bool {
	if !c.isSharded() {
		return true
	}

	level.Debug(log.Logger).Log("msg", "checking hash", "hash", hash)

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(hash))
	hash32 := hasher.Sum32()

	rs, err := c.Ring.Get(hash32, ring.Read, []ring.InstanceDesc{}, nil, nil)
	if err != nil {
		level.Error(log.Logger).Log("msg", "failed to get ring", "err", err)
		return false
	}

	if len(rs.Instances) != 1 {
		level.Error(log.Logger).Log("msg", "unexpected number of compactors in the shard (expected 1, got %d)", len(rs.Instances))
		return false
	}

	level.Debug(log.Logger).Log("msg", "checking addresses", "owning_addr", rs.Instances[0].Addr, "this_addr", c.ringLifecycler.Addr)

	return rs.Instances[0].Addr == c.ringLifecycler.Addr
}

// Combine implements common.ObjectCombiner
func (c *Compactor) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool) {
	return model.ObjectCombiner.Combine(dataEncoding, objs...)
}

// BlockRetentionForTenant implements CompactorOverrides
func (c *Compactor) BlockRetentionForTenant(tenantID string) time.Duration {
	return c.overrides.BlockRetention(tenantID)
}

func (c *Compactor) waitRingActive(ctx context.Context) error {
	for {
		// Check if the ingester is ACTIVE in the ring and our ring client
		// has detected it.
		if rs, err := c.Ring.GetAllHealthy(ring.Reporting); err == nil {
			for _, i := range rs.Instances {
				if i.GetAddr() == c.ringLifecycler.Addr && i.GetState() == ring.ACTIVE {
					return nil
				}
			}
		}

		select {
		case <-time.After(time.Second):
			// Nothing to do
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *Compactor) isSharded() bool {
	return c.cfg.ShardingRing.KVStore.Store != ""
}
