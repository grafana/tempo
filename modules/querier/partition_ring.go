package querier

import (
	"context"
	"math/rand/v2"

	"github.com/grafana/dskit/concurrency"
	"github.com/grafana/dskit/ring"
)

// forPartitionRingReplicaSets runs f, in parallel, for all ingesters in the input replicationSets.
// Return an error if any f fails for any of the input replicationSets.
func forPartitionRingReplicaSets[R any, TClient any](ctx context.Context, q *Querier, replicationSets []ring.ReplicationSet, f func(context.Context, TClient) (R, error)) ([]R, error) {
	wrappedF := func(ctx context.Context, ingester *ring.InstanceDesc) (R, error) {
		client, err := q.liveStorePool.GetClientForInstance(*ingester)
		if err != nil {
			var empty R
			return empty, err
		}

		return f(ctx, client.(TClient))
	}

	cleanup := func(_ R) {
		// Nothing to do.
	}

	quorumConfig := q.queryQuorumConfigForReplicationSets(ctx, replicationSets)

	return concurrency.ForEachJobMergeResults[ring.ReplicationSet, R](ctx, replicationSets, 0, func(ctx context.Context, set ring.ReplicationSet) ([]R, error) {
		return ring.DoUntilQuorum(ctx, set, quorumConfig, wrappedF, cleanup)
	})
}

// queryQuorumConfigForReplicationSets returns the config to use with "do until quorum" functions when running queries.
func (q *Querier) queryQuorumConfigForReplicationSets(_ context.Context, _ []ring.ReplicationSet) ring.DoUntilQuorumConfig {
	zoneSorter := queryIngesterPartitionsRingZoneSorter(q.cfg.PartitionRing.PreferredZone)

	return ring.DoUntilQuorumConfig{
		MinimizeRequests: q.cfg.PartitionRing.MinimizeRequests,
		HedgingDelay:     q.cfg.PartitionRing.MinimizeRequestsHedgingDelay,
		ZoneSorter:       zoneSorter,
		Logger:           nil, // pass a logger?
	}
}

// queryIngesterPartitionsRingZoneSorter returns a ring.ZoneSorter that should be used to sort
// ingester zones to attempt to query first, when ingest storage is enabled.
//
// The sorter gives preference to preferredZone if non empty, and then randomize the other zones.
func queryIngesterPartitionsRingZoneSorter(preferredZone string) ring.ZoneSorter {
	return func(zones []string) []string {
		// Shuffle the zones to distribute load evenly.
		if len(zones) > 2 || (preferredZone == "" && len(zones) > 1) {
			rand.Shuffle(len(zones), func(i, j int) {
				zones[i], zones[j] = zones[j], zones[i]
			})
		}

		if preferredZone != "" {
			// Give priority to the preferred zone.
			for i, z := range zones {
				if z == preferredZone {
					zones[0], zones[i] = zones[i], zones[0]
					break
				}
			}
		}

		return zones
	}
}
