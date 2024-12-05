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

	gen := NewDeterministicIDGenerator(util.FakeTenantID, 0, ts)

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

	gen = NewDeterministicIDGenerator(util.FakeTenantID, 0, ts)
	for seq := int64(0); seq < 10; seq++ {
		id := gen.NewID()
		if _, ok := firstPassIDs[id]; !ok {
			t.Errorf("ID %s not found in first pass IDs", id)
		}
	}
}

func FuzzDeterministicIDGenerator(f *testing.F) {
	f.Skip()

	f.Add(util.FakeTenantID, int64(42))
	f.Fuzz(func(t *testing.T, tenantID string, seed int64) {
		gen := NewDeterministicIDGenerator(tenantID, seed)

		for i := 0; i < 3; i++ {
			if err := generateAndParse(gen); err != nil {
				t.Errorf("Error generating and parsing ID: %v", err)
			}
		}
	})
}

func generateAndParse(g IDGenerator) error {
	id := g.NewID()
	_, err := uuid.Parse(id.String())
	return err
}

func BenchmarkDeterministicID(b *testing.B) {
	tenant := util.FakeTenantID
	ts := time.Now().UnixMilli()
	partitionID := int64(0)
	gen := NewDeterministicIDGenerator(tenant, partitionID, ts)
	for i := 0; i < b.N; i++ {
		_ = gen.NewID()
	}
}

func BenchmarkNewID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}
