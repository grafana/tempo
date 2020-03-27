package compactor

import (
	"bytes"
	"context"
	"hash/fnv"
	"time"

	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/services"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/pkg/errors"
)

const CompactorRingKey = "compactor"

type Compactor struct {
	cfg   *Config
	store storage.Store

	// Ring used for sharding compactions.
	ringLifecycler *ring.Lifecycler
	Ring           *ring.Ring
}

// New makes a new Querier.
func New(cfg Config, storeCfg storage.Config, store storage.Store) (*Compactor, error) {
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
		c.Ring = ring

		ctx := context.Background()

		level.Info(util.Logger).Log("msg", "starting ring and lifecycler")
		err = services.StartAndAwaitRunning(ctx, c.ringLifecycler)
		if err != nil {
			return nil, err
		}
		err = services.StartAndAwaitRunning(ctx, c.Ring)
		if err != nil {
			return nil, err
		}

		level.Info(util.Logger).Log("msg", "waiting to be active in the ring")
		err = c.waitRingActive(ctx)
		if err != nil {
			return nil, err
		}

		// if there is already a compactor in the ring then let's wait one poll cycle here to reduce the chance
		// of compactor collisions
		rset, err := c.Ring.GetAll()
		if err != nil {
			return nil, err
		}

		if len(rset.Ingesters) > 1 {
			level.Info(util.Logger).Log("msg", "found more than 1 ingester.  waiting one poll cycle", "waitDuration", storeCfg.Trace.MaintenanceCycle)
			time.Sleep(storeCfg.Trace.MaintenanceCycle)
		}
	}

	level.Info(util.Logger).Log("msg", "enabling compaction")
	store.EnableCompaction(cfg.Compactor, c)

	return c, nil
}

// Shutdown stops the ingester.
func (c *Compactor) Shutdown() {
	err := services.StopAndAwaitTerminated(context.Background(), c.ringLifecycler)
	if err != nil {
		level.Error(util.Logger).Log("msg", "stop lifecycler failed", "err", err)
	}
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
	// if the byte arrays are the same, we can return quickly
	hasher := fnv.New32a()

	_, _ = hasher.Write(objA)
	hashA := hasher.Sum32()
	hasher.Reset()
	_, _ = hasher.Write(objB)
	hashB := hasher.Sum32()
	if hashA == hashB {
		return objA
	}

	// hashes differ.  unmarshal and combine traces
	traceA := &tempopb.Trace{}
	traceB := &tempopb.Trace{}

	errA := proto.Unmarshal(objA, traceA)
	if errA != nil {
		level.Error(util.Logger).Log("msg", "error unsmarshaling objA", "err", errA)
	}

	errB := proto.Unmarshal(objB, traceB)
	if errB != nil {
		level.Error(util.Logger).Log("msg", "error unsmarshaling objB", "err", errB)
	}

	// if we had problems unmarshaling one or the other, return the one that marshalled successfully
	if errA != nil && errB == nil {
		return objB
	} else if errB != nil && errA == nil {
		return objA
	} else if errA != nil && errB != nil {
		// if both failed let's send back an empty trace
		level.Error(util.Logger).Log("msg", "both A and B failed to unmarshal.  returning an empty trace")
		bytes, err := proto.Marshal(&tempopb.Trace{})
		if err != nil {
			level.Error(util.Logger).Log("msg", "somehow marshalling an empty trace threw an error.", "err", err)
		}
		return bytes
	}

	// loop through every span and copy spans in B that don't exist to A
	for _, batchB := range traceB.Batches {
		notFoundSpans := batchB.Spans[:0]
		for _, spanB := range batchB.Spans {
			foundSpan := false

		A:
			for _, batchA := range traceA.Batches {
				for _, spanA := range batchA.Spans {
					if bytes.Equal(spanA.SpanId, spanB.SpanId) {
						foundSpan = true
						break A
					}
				}
			}

			// if found in A, remove from the batch
			if !foundSpan {
				notFoundSpans = append(notFoundSpans, spanB)
			}
		}

		// if there were some spans not found in A, add everything left in the batch
		if len(notFoundSpans) > 0 {
			batchB.Spans = notFoundSpans
			traceA.Batches = append(traceA.Batches, batchB)
		}
	}

	bytes, err := proto.Marshal(traceA)
	if err != nil {
		level.Error(util.Logger).Log("msg", "marshalling the combine trace threw an error.", "err", err)
		return objA
	}
	return bytes
}

func (c *Compactor) waitRingActive(ctx context.Context) error {
	for {
		// Check if the ingester is ACTIVE in the ring and our ring client
		// has detected it.
		if rs, err := c.Ring.GetAll(); err == nil {
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
