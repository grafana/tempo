package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BenchmarkTenantIndex includes reading the compressed object, decompression and
// protobuf decoding. The in-memory backend excludes network latency.
func BenchmarkTenantIndex(b *testing.B) {
	for _, tc := range []struct {
		name    string
		blocks  int
		layouts int
		cold    bool
	}{
		{"shared/100", 100, 10, false},
		{"shared/1000", 1000, 10, false},
		{"shared/10000", 10000, 10, false},
		{"cold/1000", 1000, 10, true},
		{"churn/4096", 4096, 4096, false},
		{"none/1000", 1000, 0, false},
	} {
		b.Run(tc.name, func(b *testing.B) {
			idx := tenantIndexReadFixture(tc.blocks, tc.layouts)
			data, err := idx.marshalPb()
			require.NoError(b, err)
			r := NewReader(&MockRawReader{R: data})
			ctx := context.Background()
			dedicatedColumnsCache.InvalidateAll()
			got, err := r.TenantIndex(ctx, "test-tenant")
			require.NoError(b, err)
			require.Equal(b, idx, got)
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				if tc.cold {
					b.StopTimer()
					dedicatedColumnsCache.InvalidateAll()
					b.StartTimer()
				}
				doNotOptimizeTenantIndex, err = r.TenantIndex(ctx, "test-tenant")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func tenantIndexReadFixture(blocks, layouts int) *TenantIndex {
	now := time.Unix(1_700_000_000, 0).UTC()
	rng := rand.New(rand.NewPCG(1, 2))
	idx := &TenantIndex{CreatedAt: now}
	for i := range blocks {
		var id UUID
		binary.LittleEndian.PutUint64(id[:8], rng.Uint64())
		binary.LittleEndian.PutUint64(id[8:], rng.Uint64())
		end := now.Add(-time.Duration(rng.IntN(24*60)) * time.Minute)
		meta := &BlockMeta{
			BlockID: id, TenantID: "test-tenant", Version: "vParquet5",
			StartTime: end.Add(-time.Hour), EndTime: end,
			TotalObjects: int64(50_000 + rng.IntN(100_000)), Size_: 50_000_000 + rng.Uint64N(100_000_000), TotalRecords: 100_000,
			BloomShardCount: 4, FooterSize: 4096, ReplicationFactor: 1,
		}
		if layouts > 0 {
			meta.DedicatedColumns = append(DefaultDedicatedColumns(), DedicatedColumn{
				Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString,
				Name: fmt.Sprintf("custom.attribute.%d", i%layouts),
			})
		}
		if i%5 == 0 {
			idx.CompactedMeta = append(idx.CompactedMeta, &CompactedBlockMeta{BlockMeta: *meta, CompactedTime: now})
		} else {
			idx.Meta = append(idx.Meta, meta)
		}
	}
	return idx
}
