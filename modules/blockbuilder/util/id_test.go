package util

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDeterministicID(t *testing.T) {
	ts := time.Now().UnixMilli()

	firstPassIDs := make(map[uuid.UUID]struct{})
	for seq := int64(0); seq < 10; seq++ {
		id := NewDeterministicID(ts, seq)
		firstPassIDs[id] = struct{}{}
	}

	for seq := int64(0); seq < 10; seq++ {
		id := NewDeterministicID(ts, seq)
		if _, ok := firstPassIDs[id]; !ok {
			t.Errorf("ID %s not found in first pass IDs", id)
		}
	}
}

func BenchmarkDeterministicID(b *testing.B) {
	ts := time.Now().UnixMilli()
	for i := 0; i < b.N; i++ {
		_ = NewDeterministicID(ts, int64(i))
	}
}

func BenchmarkNewID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}
