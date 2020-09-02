package compactor

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/tempo/modules/storage"
	tempo_util "github.com/grafana/tempo/pkg/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

const CompactorRingKey = "compactor"

type Compactor struct {
	services.Service

	cfg   *Config
	store storage.Store

	// Ring used for sharding compactions.
	ringLifecycler *ring.Lifecycler
	Ring           *ring.Ring

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

// New makes a new Querier.
func New(cfg Config, store storage.Store) (*Compactor, error) {
	c := &Compactor{
		cfg:   &cfg,
		store: store,
	}

	subservices := []services.Service(nil)
	if c.cfg.ShardingEnabled {
		lifecyclerCfg := c.cfg.ShardingRing.ToLifecyclerConfig()
		lifecycler, err := ring.NewLifecycler(lifecyclerCfg, ring.NewNoopFlushTransferer(), "compactor", CompactorRingKey, false, prometheus.DefaultRegisterer)
		if err != nil {
			return nil, errors.Wrap(err, "unable to initialize compactor ring lifecycler")
		}
		c.ringLifecycler = lifecycler
		subservices = append(subservices, c.ringLifecycler)

		ring, err := ring.New(lifecyclerCfg.RingConfig, "compactor", CompactorRingKey, prometheus.DefaultRegisterer)
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

		level.Info(util.Logger).Log("msg", "waiting to be active in the ring")
		err = c.waitRingActive(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Compactor) running(ctx context.Context) error {
	go func() {
		level.Info(util.Logger).Log("msg", "waiting one poll cycle", "waitDuration", c.cfg.WaitOnStartup)
		time.Sleep(c.cfg.WaitOnStartup)

		level.Info(util.Logger).Log("msg", "enabling compaction")
		c.store.EnableCompaction(c.cfg.Compactor, c)
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

func (c *Compactor) Owns(hash string) bool {
	if c.cfg.ShardingEnabled {
		hasher := fnv.New32a()
		_, _ = hasher.Write([]byte(hash))
		hash32 := hasher.Sum32()

		// Check whether this compactor instance owns the user.
		rs, err := c.Ring.Get(hash32, ring.Read, []ring.IngesterDesc{})
		if err != nil {
			level.Error(util.Logger).Log("msg", "failed to get ring", "err", err)
			return false
		}

		if len(rs.Ingesters) != 1 {
			level.Error(util.Logger).Log("msg", "unexpected number of compactors in the shard (expected 1, got %d)", len(rs.Ingesters))
			return false
		}

		return rs.Ingesters[0].Addr == c.ringLifecycler.Addr
	}

	return true
}

func (c *Compactor) Combine(objA []byte, objB []byte) []byte {
	return tempo_util.CombineTraces(objA, objB)
}

func (c *Compactor) waitRingActive(ctx context.Context) error {
	for {
		// Check if the ingester is ACTIVE in the ring and our ring client
		// has detected it.
		if rs, err := c.Ring.GetAll(ring.Reporting); err == nil {
			for _, i := range rs.Ingesters {
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
