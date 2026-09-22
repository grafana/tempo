package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type discardIndexWriter struct{ MockRawWriter }

func (*discardIndexWriter) Write(_ context.Context, _ string, _ KeyPath, r io.Reader, _ int64, _ *CacheInfo) error {
	_, err := io.Copy(io.Discard, r)
	return err
}

// BenchmarkWriteTenantIndex includes both compressed index formats and consumes
// each complete object in memory, excluding network latency.
func BenchmarkWriteTenantIndex(b *testing.B) {
	for _, tc := range []struct {
		name            string
		blocks, layouts int
		suffix          string
		single          bool
		all             bool
	}{
		{"shared100", 100, 10, "", false, false},
		{"shared1000", 1000, 10, "", false, false},
		{"shared10000", 10000, 10, "", false, false},
		{"churn4096", 4096, 4096, "", false, false},
		{"none1000", 1000, 0, "", false, false},
		{"escaped1000", 1000, 10, "\"\\\n☃", false, false},
		{"lateEscape1000", 1000, 10, strings.Repeat("x", 4096) + "\"", false, false},
		{"longASCII1000", 1000, 10, strings.Repeat("x", 65536), false, false},
		{"escapedOnly1000", 1000, 10, "\"\\\n☃", true, false},
		{"longOnly1000", 1000, 10, strings.Repeat("x", 65536), true, false},
		{"escapedAll1000", 1000, 10, "\"\\\n☃", false, true},
		{"name256Plain1000", 1000, 10, "", false, true},
		{"name256Escaped1000", 1000, 10, "\"", false, true},
		{"name32Plain1000", 1000, 10, "", false, true},
		{"name32Escaped1000", 1000, 10, "\"", false, true},
	} {
		b.Run(tc.name, func(b *testing.B) {
			idx := tenantIndexWriteFixture(tc.blocks, tc.layouts)
			alter := func(m *BlockMeta) {
				cols := m.DedicatedColumns.Columns()
				defer func() { m.DedicatedColumns = NewDedicatedColumnLayout(cols) }()
				if strings.HasPrefix(tc.name, "name32") || strings.HasPrefix(tc.name, "name256") {
					limit := 32
					if strings.HasPrefix(tc.name, "name256") {
						limit = 256
					}
					for i := range cols {
						name := cols[i].Name
						cols[i].Name = name + strings.Repeat("x", limit-len(name)-len(tc.suffix)) + tc.suffix
					}
					return
				}
				if tc.single {
					cols = cols[len(cols)-1:]
				}
				if tc.suffix != "" {
					if tc.all {
						for i := range cols {
							cols[i].Name += tc.suffix
						}
					} else {
						cols[len(cols)-1].Name += tc.suffix
					}
				}
			}
			for _, m := range idx.Meta {
				alter(m)
			}
			for _, m := range idx.CompactedMeta {
				alter(&m.BlockMeta)
			}

			w := NewWriter(&discardIndexWriter{})
			require.NoError(b, w.WriteTenantIndex(context.Background(), "test-tenant", idx.Meta, idx.CompactedMeta))
			b.ReportAllocs()
			for b.Loop() {
				if err := w.WriteTenantIndex(context.Background(), "test-tenant", idx.Meta, idx.CompactedMeta); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func tenantIndexWriteFixture(blocks, layouts int) *TenantIndex {
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
			meta.DedicatedColumns = NewDedicatedColumnLayout(append(DefaultDedicatedColumns(), DedicatedColumn{
				Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString,
				Name: fmt.Sprintf("custom.attribute.%d", i%layouts),
			}))
		}
		if i%5 == 0 {
			idx.CompactedMeta = append(idx.CompactedMeta, &CompactedBlockMeta{BlockMeta: *meta, CompactedTime: now})
		} else {
			idx.Meta = append(idx.Meta, meta)
		}
	}
	return idx
}
