package backend

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	// globals to avoid compiler optimizations
	doNotOptimizeBytes       []byte
	doNotOptimizeTenantIndex *TenantIndex
)

func BenchmarkIndexMarshal(b *testing.B) {
	b.Run("format=json", func(b *testing.B) {
		for _, numBlocks := range []int{100, 1000, 10000} {
			b.Run(fmt.Sprintf("blocks=%d", numBlocks), func(b *testing.B) {
				dedicatedColumnsCache.Purge()
				idx := makeTestTenantIndex(numBlocks)
				for b.Loop() {
					doNotOptimizeBytes, _ = idx.marshal()
				}
			})
		}
	})

	b.Run("format=proto", func(b *testing.B) {
		for _, numBlocks := range []int{100, 1000, 10000} {
			b.Run(fmt.Sprintf("blocks=%d", numBlocks), func(b *testing.B) {
				dedicatedColumnsCache.Purge()
				idx := makeTestTenantIndex(numBlocks)
				for b.Loop() {
					doNotOptimizeBytes, _ = idx.marshalPb()
				}
			})
		}
	})
}

func BenchmarkIndexUnmarshal(b *testing.B) {
	b.Run("format=json", func(b *testing.B) {
		for _, numBlocks := range []int{100, 1000, 10000} {
			b.Run(fmt.Sprintf("blocks=%d", numBlocks), func(b *testing.B) {
				dedicatedColumnsCache.Purge()
				idx := makeTestTenantIndex(numBlocks)
				idxBuf, err := idx.marshal()
				require.NoError(b, err)
				for b.Loop() {
					doNotOptimizeTenantIndex = &TenantIndex{}
					_ = doNotOptimizeTenantIndex.unmarshal(idxBuf)
				}
			})
		}
	})

	b.Run("format=proto", func(b *testing.B) {
		for _, numBlocks := range []int{100, 1000, 10000} {
			b.Run(fmt.Sprintf("blocks=%d", numBlocks), func(b *testing.B) {
				dedicatedColumnsCache.Purge()
				idx := makeTestTenantIndex(numBlocks)
				idxBuf, err := idx.marshal()
				require.NoError(b, err)
				for b.Loop() {
					doNotOptimizeTenantIndex = &TenantIndex{}
					_ = doNotOptimizeTenantIndex.unmarshalPb(idxBuf)
				}
			})
		}
	})
}

func makeTestTenantIndex(numBlocks int) *TenantIndex {
	const numDistinctDedicatedCols = 10

	var (
		maxSupportedSpanColumns     = maxSupportedColumns[DedicatedColumnTypeString][DedicatedColumnScopeSpan]
		maxSupportedResourceColumns = maxSupportedColumns[DedicatedColumnTypeString][DedicatedColumnScopeResource]
	)

	dedicatedCols := make([]DedicatedColumns, 0, numDistinctDedicatedCols)
	for range numDistinctDedicatedCols {
		num := 0
		cols := make([]DedicatedColumn, 0, maxSupportedSpanColumns+maxSupportedResourceColumns)
		for range maxSupportedSpanColumns {
			num += rand.IntN(10)
			cols = append(cols, DedicatedColumn{
				Scope: DedicatedColumnScopeSpan,
				Name:  fmt.Sprintf("ded-span-%d", num),
				Type:  DedicatedColumnTypeString,
			})
		}
		for range maxSupportedResourceColumns {
			num += rand.IntN(10)
			cols = append(cols, DedicatedColumn{
				Scope: DedicatedColumnScopeResource,
				Name:  fmt.Sprintf("ded-res-%d", num),
				Type:  DedicatedColumnTypeString,
			})
		}
		dedicatedCols = append(dedicatedCols, cols)
	}

	blocks := make([]*BlockMeta, 0, numBlocks)
	compactedBlocks := make([]*CompactedBlockMeta, 0, numBlocks)
	for i := range numBlocks {
		meta := NewBlockMeta("test-tenant", uuid.New(), "vParquet4", EncNone, "")
		meta.DedicatedColumns = dedicatedCols[i%numDistinctDedicatedCols]
		blocks = append(blocks, meta)

		compactedMeta := &CompactedBlockMeta{
			BlockMeta:     *NewBlockMeta("test-tenant", uuid.New(), "vParquet4", EncNone, ""),
			CompactedTime: time.Now(),
		}
		compactedMeta.DedicatedColumns = dedicatedCols[i%numDistinctDedicatedCols]
		compactedBlocks = append(compactedBlocks, compactedMeta)
	}

	return newTenantIndex(blocks, compactedBlocks)
}
