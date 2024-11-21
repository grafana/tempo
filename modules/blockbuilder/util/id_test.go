package util

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

func TestDeterministicIDGenerator(t *testing.T) {
	ts := time.Now().UnixMilli()

	gen := NewDeterministicIDGenerator(ts)

	firstPassIDs := make(map[backend.UUID]struct{})
	for seq := int64(0); seq < 10; seq++ {
		id := gen.NewID()
		firstPassIDs[id] = struct{}{}
	}

	gen = NewDeterministicIDGenerator(ts)
	for seq := int64(0); seq < 10; seq++ {
		id := gen.NewID()
		if _, ok := firstPassIDs[id]; !ok {
			t.Errorf("ID %s not found in first pass IDs", id)
		}
	}
}

func BenchmarkDeterministicID(b *testing.B) {
	ts := time.Now().UnixMilli()
	gen := NewDeterministicIDGenerator(ts)
	for i := 0; i < b.N; i++ {
		_ = gen.NewID()
	}
}

func BenchmarkNewID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}
