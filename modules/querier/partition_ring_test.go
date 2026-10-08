package querier

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/grafana/dskit/ring"
	"github.com/stretchr/testify/require"
)

func TestLiveStoreReplicationSets(t *testing.T) {
	const lookback = 30 * time.Minute
	tests := []struct {
		name      string
		state     ring.PartitionState
		age       time.Duration
		owner     bool
		lookback  time.Duration
		wantIDs   []string
		wantError error
	}{
		{name: "active is retained regardless of age", state: ring.PartitionActive, age: time.Hour, owner: true, lookback: lookback, wantIDs: []string{"active", "other"}},
		{name: "recent inactive is retained", state: ring.PartitionInactive, age: 29 * time.Minute, owner: true, lookback: lookback, wantIDs: []string{"active", "other"}},
		{name: "inactive at cutoff is retained", state: ring.PartitionInactive, age: lookback, owner: true, lookback: lookback, wantIDs: []string{"active", "other"}},
		{name: "old ownerless inactive is excluded", state: ring.PartitionInactive, age: lookback + time.Second, lookback: lookback, wantIDs: []string{"active"}},
		{name: "pending is excluded", state: ring.PartitionPending, lookback: lookback, wantIDs: []string{"active"}},
		{name: "ownerless active still fails", state: ring.PartitionActive, lookback: lookback, wantError: ring.ErrTooManyUnhealthyInstances},
		{name: "ownerless recent inactive still fails", state: ring.PartitionInactive, age: time.Minute, lookback: lookback, wantError: ring.ErrTooManyUnhealthyInstances},
		{name: "disabled filtering retains old inactive", state: ring.PartitionInactive, age: time.Hour, owner: true, wantIDs: []string{"active", "other"}},
		{name: "disabled filtering preserves ownerless error", state: ring.PartitionInactive, age: time.Hour, wantError: ring.ErrTooManyUnhealthyInstances},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().Truncate(time.Second)
			desc := ring.NewPartitionRingDesc()
			desc.AddPartition(0, ring.PartitionActive, now.Add(-time.Hour))
			desc.AddOrUpdateOwner("active", ring.OwnerActive, 0, now)
			desc.AddPartition(1, tt.state, now.Add(-tt.age))
			instances := testInstanceRingReader{
				"active": {Id: "active", State: ring.ACTIVE, Timestamp: now.Unix(), Zone: "zone-a"},
			}
			if tt.owner {
				desc.AddOrUpdateOwner("other", ring.OwnerActive, 1, now)
				instances["other"] = ring.InstanceDesc{Id: "other", State: ring.ACTIVE, Timestamp: now.Unix(), Zone: "zone-a"}
			}
			partitions, err := ring.NewPartitionRing(*desc)
			require.NoError(t, err)
			q := &Querier{
				cfg:           Config{PartitionRing: PartitionRingConfig{ReadLookbackPeriod: tt.lookback}},
				partitionRing: ring.NewPartitionInstanceRing(testPartitionRingReader{partitions}, instances, time.Minute),
			}

			sets, err := q.liveStoreReplicationSets(now)
			if tt.wantError != nil {
				require.ErrorIs(t, err, tt.wantError)
				require.Nil(t, sets)
				return
			}
			require.NoError(t, err)
			var ids []string
			for _, set := range sets {
				require.Len(t, set.Instances, 1)
				ids = append(ids, set.Instances[0].Id)
			}
			sort.Strings(ids)
			require.Equal(t, tt.wantIDs, ids)
		})
	}
}

func TestLiveStoreReplicationSetsEmptyRing(t *testing.T) {
	for _, filteredEmpty := range []bool{false, true} {
		name := "empty ring"
		if filteredEmpty {
			name = "all partitions filtered out"
		}
		t.Run(name, func(t *testing.T) {
			now := time.Now()
			desc := ring.NewPartitionRingDesc()
			if filteredEmpty {
				desc.AddPartition(0, ring.PartitionInactive, now.Add(-time.Hour))
			}
			partitions, err := ring.NewPartitionRing(*desc)
			require.NoError(t, err)
			q := &Querier{
				cfg:           Config{PartitionRing: PartitionRingConfig{ReadLookbackPeriod: 30 * time.Minute}},
				partitionRing: ring.NewPartitionInstanceRing(testPartitionRingReader{partitions}, testInstanceRingReader{}, time.Minute),
			}

			sets, err := q.liveStoreReplicationSets(now)
			require.ErrorIs(t, err, ring.ErrEmptyRing)
			require.Nil(t, sets)
		})
	}
}

type testPartitionRingReader struct {
	partitions *ring.PartitionRing
}

func (r testPartitionRingReader) PartitionRing() *ring.PartitionRing {
	return r.partitions
}

type testInstanceRingReader map[string]ring.InstanceDesc

func (r testInstanceRingReader) GetInstance(id string) (ring.InstanceDesc, error) {
	instance, ok := r[id]
	if !ok {
		return ring.InstanceDesc{}, errors.New("instance not found")
	}
	return instance, nil
}

func (r testInstanceRingReader) InstancesCount() int {
	return len(r)
}
