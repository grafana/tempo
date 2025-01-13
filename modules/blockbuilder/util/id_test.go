package util

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

func TestDeterministicIDGenerator(t *testing.T) {
	ts := time.Now().UnixMilli()

	gen := NewDeterministicIDGenerator(util.FakeTenantID, 0, uint64(ts))

	firstPassIDs := make(map[backend.UUID]struct{})
	for seq := int64(0); seq < 10; seq++ {
		id := gen.NewID()
		firstPassIDs[id] = struct{}{}
	}

	// Verify that that UUIDs are valid
	for id := range firstPassIDs {
		_, err := uuid.Parse(id.String())
		assert.NoError(t, err)
	}

	gen = NewDeterministicIDGenerator(util.FakeTenantID, 0, uint64(ts))
	for seq := int64(0); seq < 10; seq++ {
		id := gen.NewID()
		if _, ok := firstPassIDs[id]; !ok {
			t.Errorf("ID %s not found in first pass IDs", id)
		}
	}
}

func TestDeterministicIDGeneratorWithDifferentTenants(t *testing.T) {
	ts := time.Now().UnixMilli()
	seed := uint64(42)

	gen1 := NewDeterministicIDGenerator("tenant-1", seed, uint64(ts))
	gen2 := NewDeterministicIDGenerator("tenant-2", seed, uint64(ts))

	for i := 0; i < 10; i++ {
		assert.NotEqualf(t, gen1.NewID(), gen2.NewID(), "IDs should be different")
	}
}

func FuzzDeterministicIDGenerator(f *testing.F) {
	f.Skip()

	f.Add(util.FakeTenantID, uint64(42), uint64(100))
	f.Fuzz(func(t *testing.T, tenantID string, seed1, seed2 uint64) {
		gen := NewDeterministicIDGenerator(tenantID, seed1, seed2)

		for i := 0; i < 3; i++ {
			id := gen.NewID()
			_, err := uuid.Parse(id.String())
			if err != nil {
				t.Fatalf("failed to parse UUID: %v", err)
			}
		}
	})
}

func BenchmarkDeterministicID(b *testing.B) {
	tenant := util.FakeTenantID
	ts := time.Now().UnixMilli()
	partitionID := uint64(0)
	gen := NewDeterministicIDGenerator(tenant, partitionID, uint64(ts))
	for i := 0; i < b.N; i++ {
		_ = gen.NewID()
	}
}

func BenchmarkNewID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}
