package compactor

import (
	"hash/fnv"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/tempo/pkg/storage"
	"github.com/pkg/errors"
)

const CompactorRingKey = "compactor"

type Compactor struct {
	cfg   *Config
	store storage.Store

	// Ring used for sharding compactions.
	ringLifecycler *ring.Lifecycler
	ring           *ring.Ring
}

// New makes a new Querier.
func New(cfg Config, store storage.Store) (*Compactor, error) {
	c := &Compactor{
		cfg:   &cfg,
		store: store,
	}

	if c.cfg.ShardingEnabled {
		lifecyclerCfg := c.cfg.ShardingRing.ToLifecyclerConfig()
		lifecycler, err := ring.NewLifecycler(lifecyclerCfg, ring.NewNoopFlushTransferer(), "compactor", CompactorRingKey, false)
		if err != nil {
			return nil, errors.Wrap(err, "unable to initialize compactor ring lifecycler")
		}

		c.ringLifecycler = lifecycler
		ring, err := ring.New(lifecyclerCfg.RingConfig, "compactor", CompactorRingKey)
		if err != nil {
			return nil, errors.Wrap(err, "unable to initialize compactor ring")
		}

		c.ring = ring
	}

	store.EnableCompaction(cfg.Compactor, c)

	return c, nil
}

func (c *Compactor) Owns(hash string) bool {
	if c.cfg.ShardingEnabled {
		hasher := fnv.New32a()
		_, _ = hasher.Write([]byte(hash))
		hash32 := hasher.Sum32()

		// Check whether this compactor instance owns the user.
		rs, err := c.ring.Get(hash32, ring.Read, []ring.IngesterDesc{})
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
	if len(objA) > len(objB) {
		return objA
	}

	return objB
}
