package backend

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// indexFormat abstracts the encodings so that every benchmark compares them on identical data.
type indexFormat struct {
	name      string
	marshal   func(*TenantIndex) ([]byte, error)
	unmarshal func(*TenantIndex, []byte) error
}

var indexFormats = []indexFormat{
	{"json-gzip", (*TenantIndex).marshal, (*TenantIndex).unmarshal},
	{"proto-zstd", (*TenantIndex).marshalPb, (*TenantIndex).unmarshalPb},
	{"proto-raw", func(i *TenantIndex) ([]byte, error) { return i.Marshal() }, func(i *TenantIndex, b []byte) error { return i.Unmarshal(b) }},
	{"v2-zstd", (*TenantIndex).marshalV2Zstd, (*TenantIndex).unmarshalV2Zstd},
	{"v2-raw", (*TenantIndex).marshalV2, (*TenantIndex).unmarshalV2},
}

var benchBlockCounts = []int{100, 1000, 10000}

// makeRealisticTenantIndex is makeTestTenantIndex with the remaining BlockMeta fields filled in
// with production-like values. The zero values in makeTestTenantIndex flatter every format.
func makeRealisticTenantIndex(numBlocks int) *TenantIndex {
	return fillRealistic(makeTestTenantIndex(numBlocks))
}

func fillRealistic(idx *TenantIndex) *TenantIndex {
	rng := rand.New(rand.NewPCG(1, 2))
	base := time.Now().Add(-24 * time.Hour).Truncate(time.Second)

	fill := func(m *BlockMeta) {
		m.Version = "vParquet5"
		m.StartTime = base.Add(time.Duration(rng.IntN(86400)) * time.Second)
		m.EndTime = m.StartTime.Add(time.Duration(rng.IntN(600)) * time.Second)
		m.TotalObjects = rng.Int64N(5_000_000)
		m.Size_ = uint64(rng.Int64N(2 << 30))
		m.CompactionLevel = uint32(rng.IntN(4))
		m.TotalRecords = uint32(rng.IntN(5000))
		m.FooterSize = uint32(20000 + rng.IntN(20000))
		m.BloomShardCount = uint32(1 + rng.IntN(8))
		m.IndexPageSize = 250 * 1024
		if rng.IntN(5) == 0 {
			m.ReplicationFactor = MetricsGeneratorReplicationFactor
		}
	}
	for _, m := range idx.Meta {
		fill(m)
	}
	for _, m := range idx.CompactedMeta {
		fill(&m.BlockMeta)
		m.CompactedTime = m.EndTime.Add(time.Hour)
	}
	return idx
}

func BenchmarkTenantIndexV2Marshal(b *testing.B) {
	for _, f := range indexFormats {
		b.Run("format="+f.name, func(b *testing.B) {
			for _, n := range benchBlockCounts {
				b.Run(fmt.Sprintf("blocks=%d", n), func(b *testing.B) {
					idx := makeRealisticTenantIndex(n)
					b.ReportAllocs()
					for b.Loop() {
						var err error
						doNotOptimizeBytes, err = f.marshal(idx)
						require.NoError(b, err)
					}
				})
			}
		})
	}
}

// BenchmarkTenantIndexV2Unmarshal clears the dedicated columns cache each iteration so that
// json and proto pay for parsing, as they do on the first read after a restart.
func BenchmarkTenantIndexV2Unmarshal(b *testing.B) {
	for _, f := range indexFormats {
		b.Run("format="+f.name, func(b *testing.B) {
			for _, n := range benchBlockCounts {
				b.Run(fmt.Sprintf("blocks=%d", n), func(b *testing.B) {
					buf, err := f.marshal(makeRealisticTenantIndex(n))
					require.NoError(b, err)
					b.ReportAllocs()
					for b.Loop() {
						dedicatedColumnsCache.InvalidateAll()
						doNotOptimizeTenantIndex = &TenantIndex{}
						require.NoError(b, f.unmarshal(doNotOptimizeTenantIndex, buf))
					}
				})
			}
		})
	}
}

// BenchmarkTenantIndexV2UnmarshalWarm leaves the dedicated columns cache populated, which is
// the steady state for the json and proto formats.
func BenchmarkTenantIndexV2UnmarshalWarm(b *testing.B) {
	for _, f := range indexFormats {
		b.Run("format="+f.name, func(b *testing.B) {
			for _, n := range benchBlockCounts {
				b.Run(fmt.Sprintf("blocks=%d", n), func(b *testing.B) {
					buf, err := f.marshal(makeRealisticTenantIndex(n))
					require.NoError(b, err)
					require.NoError(b, f.unmarshal(&TenantIndex{}, buf)) // warm the cache
					b.ReportAllocs()
					for b.Loop() {
						doNotOptimizeTenantIndex = &TenantIndex{}
						require.NoError(b, f.unmarshal(doNotOptimizeTenantIndex, buf))
					}
				})
			}
		})
	}
}

// BenchmarkTenantIndexV2Size reports the encoded size of each format. The time per op is
// irrelevant: read the bytes and bytes/block metrics.
func BenchmarkTenantIndexV2Size(b *testing.B) {
	for _, f := range indexFormats {
		b.Run("format="+f.name, func(b *testing.B) {
			for _, n := range benchBlockCounts {
				b.Run(fmt.Sprintf("blocks=%d", n), func(b *testing.B) {
					idx := makeRealisticTenantIndex(n)
					var size int
					for b.Loop() {
						buf, err := f.marshal(idx)
						require.NoError(b, err)
						size = len(buf)
					}
					// live and compacted blocks are both encoded
					b.ReportMetric(float64(size), "bytes")
					b.ReportMetric(float64(size)/float64(2*n), "bytes/block")
				})
			}
		})
	}
}

// BenchmarkTenantIndexV2Retained reports what stays resident after decoding: the live heap
// once the decoded index is held and all transient garbage (the compressed and raw buffers,
// intermediate messages) has been collected. This is what a long running poller carries per tenant.
func BenchmarkTenantIndexV2Retained(b *testing.B) {
	heap := func() (bytes, objects uint64) {
		// twice, so that sync.Pool contents (zstd decoders) are dropped from both readings
		runtime.GC()
		runtime.GC()
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		return ms.HeapAlloc, ms.HeapObjects
	}

	for _, f := range indexFormats {
		// Compression doesn't change what is retained, and the pooled zstd decoders make the
		// heap readings noisy.
		if strings.HasSuffix(f.name, "zstd") || strings.HasSuffix(f.name, "gzip") {
			continue
		}
		b.Run("format="+f.name, func(b *testing.B) {
			for _, n := range benchBlockCounts {
				b.Run(fmt.Sprintf("blocks=%d", n), func(b *testing.B) {
					buf, err := f.marshal(makeRealisticTenantIndex(n))
					require.NoError(b, err)

					var totalBytes, totalObjects float64
					for b.Loop() {
						b.StopTimer()
						doNotOptimizeTenantIndex = nil
						dedicatedColumnsCache.InvalidateAll()
						beforeBytes, beforeObjects := heap()

						idx := &TenantIndex{}
						require.NoError(b, f.unmarshal(idx, buf))

						afterBytes, afterObjects := heap()
						runtime.KeepAlive(idx)
						doNotOptimizeTenantIndex = idx
						b.StartTimer()

						totalBytes += float64(afterBytes) - float64(beforeBytes)
						totalObjects += float64(afterObjects) - float64(beforeObjects)
					}
					b.ReportMetric(totalBytes/float64(b.N)/float64(2*n), "retained-B/block")
					b.ReportMetric(totalObjects/float64(b.N)/float64(2*n), "retained-objs/block")
				})
			}
		})
	}
}

// makeWideSchemaTenantIndex builds numBlocks live and numBlocks compacted blocks spread over
// numSchemas distinct dedicated column layouts, each with colsPerScope columns at both the
// span and the resource level. Wide layouts are where hoisting them out of every block pays off.
func makeWideSchemaTenantIndex(numBlocks, numSchemas, colsPerScope int) *TenantIndex {
	layouts := make([]DedicatedColumns, numSchemas)
	for i := range layouts {
		cols := make(DedicatedColumns, 0, 2*colsPerScope)
		for _, scope := range []DedicatedColumnScope{DedicatedColumnScopeSpan, DedicatedColumnScopeResource} {
			for j := range colsPerScope {
				cols = append(cols, DedicatedColumn{
					Scope: scope,
					Name:  fmt.Sprintf("schema%d.%s.attribute-%d", i, scope, j),
					Type:  DedicatedColumnTypeString,
				})
			}
		}
		layouts[i] = cols
	}

	blocks := make([]*BlockMeta, 0, numBlocks)
	compacted := make([]*CompactedBlockMeta, 0, numBlocks)
	for i := range numBlocks {
		m := NewBlockMeta("test-tenant", uuid.New(), "vParquet5")
		m.DedicatedColumns = layouts[i%numSchemas]
		blocks = append(blocks, m)

		c := &CompactedBlockMeta{BlockMeta: *NewBlockMeta("test-tenant", uuid.New(), "vParquet5"), CompactedTime: time.Now()}
		c.DedicatedColumns = layouts[i%numSchemas]
		compacted = append(compacted, c)
	}
	return fillRealistic(newTenantIndex(blocks, compacted, nil))
}

type wideScenario struct {
	name                             string
	blocks, schemas, columnsPerScope int
}

var wideScenarios = []wideScenario{
	{"blocks=100000/schemas=10/cols=20+20", 100_000, 10, 20},
	{"blocks=100000/schemas=10/cols=50+50", 100_000, 10, 50},
}

// The wide benchmarks only compare the two formats that are stored: proto-zstd and v2-zstd.
var wideFormats = []indexFormat{indexFormats[1], indexFormats[3]}

func BenchmarkTenantIndexV2WideUnmarshal(b *testing.B) {
	for _, sc := range wideScenarios {
		for _, f := range wideFormats {
			b.Run(sc.name+"/format="+f.name, func(b *testing.B) {
				buf, err := f.marshal(makeWideSchemaTenantIndex(sc.blocks, sc.schemas, sc.columnsPerScope))
				require.NoError(b, err)
				b.ReportAllocs()
				for b.Loop() {
					dedicatedColumnsCache.InvalidateAll()
					doNotOptimizeTenantIndex = &TenantIndex{}
					require.NoError(b, f.unmarshal(doNotOptimizeTenantIndex, buf))
				}
			})
		}
	}
}

func BenchmarkTenantIndexV2WideMarshal(b *testing.B) {
	for _, sc := range wideScenarios {
		for _, f := range wideFormats {
			b.Run(sc.name+"/format="+f.name, func(b *testing.B) {
				idx := makeWideSchemaTenantIndex(sc.blocks, sc.schemas, sc.columnsPerScope)
				b.ReportAllocs()
				for b.Loop() {
					var err error
					doNotOptimizeBytes, err = f.marshal(idx)
					require.NoError(b, err)
				}
			})
		}
	}
}

// BenchmarkTenantIndexV2WideSize reports bytes: the stored (zstd) size, and the raw size
// that has to be held in memory while decoding.
func BenchmarkTenantIndexV2WideSize(b *testing.B) {
	for _, sc := range wideScenarios {
		b.Run(sc.name, func(b *testing.B) {
			idx := makeWideSchemaTenantIndex(sc.blocks, sc.schemas, sc.columnsPerScope)
			for b.Loop() {
			}
			// after the loop, which resets metrics
			for _, f := range []indexFormat{indexFormats[1], indexFormats[2], indexFormats[3], indexFormats[4]} {
				buf, err := f.marshal(idx)
				require.NoError(b, err)
				b.ReportMetric(float64(len(buf)), f.name+"-bytes")
			}
		})
	}
}

// BenchmarkTenantIndexV2WideRetained is BenchmarkTenantIndexV2Retained for wide schemas.
func BenchmarkTenantIndexV2WideRetained(b *testing.B) {
	heap := func() (bytes, objects uint64) {
		runtime.GC()
		runtime.GC()
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		return ms.HeapAlloc, ms.HeapObjects
	}

	for _, sc := range wideScenarios {
		for _, f := range []indexFormat{indexFormats[2], indexFormats[4]} {
			b.Run(sc.name+"/format="+f.name, func(b *testing.B) {
				buf, err := f.marshal(makeWideSchemaTenantIndex(sc.blocks, sc.schemas, sc.columnsPerScope))
				require.NoError(b, err)

				var totalBytes, totalObjects float64
				for b.Loop() {
					b.StopTimer()
					doNotOptimizeTenantIndex = nil
					dedicatedColumnsCache.InvalidateAll()
					beforeBytes, beforeObjects := heap()

					idx := &TenantIndex{}
					require.NoError(b, f.unmarshal(idx, buf))

					afterBytes, afterObjects := heap()
					runtime.KeepAlive(idx)
					doNotOptimizeTenantIndex = idx
					b.StartTimer()

					totalBytes += float64(afterBytes) - float64(beforeBytes)
					totalObjects += float64(afterObjects) - float64(beforeObjects)
				}
				b.ReportMetric(totalBytes/float64(b.N)/float64(2*sc.blocks), "retained-B/block")
				b.ReportMetric(totalObjects/float64(b.N)/float64(2*sc.blocks), "retained-objs/block")
			})
		}
	}
}
