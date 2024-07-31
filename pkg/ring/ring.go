package ring

import (
	"fmt"
	"time"

	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/v2/pkg/usagestats"
	"github.com/grafana/tempo/v2/pkg/util/log"
)

var (
	statReplicationFactor = usagestats.NewInt("ring_replication_factor")
	statKvStore           = usagestats.NewString("ring_kv_store")
)

// New creates a new distributed consistent hash ring.  It shadows the cortex
// ring.New method so we can use our own replication strategy for repl factor = 2
func New(cfg ring.Config, name, key string, reg prometheus.Registerer) (*ring.Ring, error) {
	reg = prometheus.WrapRegistererWithPrefix("tempo_", reg)

	statReplicationFactor.Set(int64(cfg.ReplicationFactor))
	statKvStore.Set(cfg.KVStore.Store)

	if cfg.ReplicationFactor == 2 {
		return newEventuallyConsistentRing(cfg, name, key, reg)
	}

	return ring.New(cfg, name, key, log.Logger, reg)
}

func newEventuallyConsistentRing(cfg ring.Config, name, key string, reg prometheus.Registerer) (*ring.Ring, error) {
	codec := ring.GetCodec()
	// Suffix all client names with "-ring" to denote this kv client is used by the ring
	store, err := kv.NewClient(
		cfg.KVStore,
		codec,
		kv.RegistererWithKVName(reg, name+"-ring"),
		log.Logger,
	)
	if err != nil {
		return nil, err
	}

	return ring.NewWithStoreClientAndStrategy(cfg, name, key, store, &EventuallyConsistentStrategy{}, reg, log.Logger)
}

// EventuallyConsistentStrategy represents a repl strategy with a consistency of 1 on read and
// write.  Note this is NOT strongly consistent!  It is _eventually_ consistent :)
type EventuallyConsistentStrategy struct{}

// Filter decides, given the set of ingesters eligible for a key,
// which ingesters you will try and write to and how many failures you will
// tolerate.
// - Filters out dead ingesters so the one doesn't even try to write to them.
// - Checks there is enough ingesters for an operation to succeed.
// The ingesters argument may be overwritten.
func (s *EventuallyConsistentStrategy) Filter(ingesters []ring.InstanceDesc, op ring.Operation, _ int, heartbeatTimeout time.Duration, _ bool) ([]ring.InstanceDesc, int, error) {
	minSuccess := 1

	// Skip those that have not heartbeated in a while. NB these are still
	// included in the calculation of minSuccess, so if too many failed ingesters
	// will cause the whole write to fail.
	for i := 0; i < len(ingesters); {
		if ingesters[i].IsHealthy(op, heartbeatTimeout, time.Now()) {
			i++
		} else {
			ingesters = append(ingesters[:i], ingesters[i+1:]...)
		}
	}

	// This is just a shortcut - if there are not minSuccess available ingesters,
	// after filtering out dead ones, don't even bother trying.
	if len(ingesters) < minSuccess {
		err := fmt.Errorf("at least %d live replicas required, could only find %d",
			minSuccess, len(ingesters))
		return nil, 0, err
	}

	return ingesters, len(ingesters) - minSuccess, nil
}

func (s *EventuallyConsistentStrategy) ShouldExtendReplicaSet(ingester ring.InstanceDesc, op ring.Operation) bool {
	// We do not want to Write to Ingesters that are not ACTIVE, but we do want
	// to write the extra replica somewhere.  So we increase the size of the set
	// of replicas for the key. This means we have to also increase the
	// size of the replica set for read, but we can read from Leaving ingesters,
	// so don't skip it in this case.
	// NB dead ingester will be filtered later by DefaultReplicationStrategy.Filter().
	if op == ring.Write && ingester.State != ring.ACTIVE {
		return true
	} else if op == ring.Read && (ingester.State != ring.ACTIVE && ingester.State != ring.LEAVING) {
		return true
	}

	return false
}
