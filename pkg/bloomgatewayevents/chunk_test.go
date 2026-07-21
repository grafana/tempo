package bloomgatewayevents

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

// idsOfLen returns n distinct byte slices of the given size (must be >= 4),
// each carrying its index in its trailing 4 bytes so pass-through mismatches
// (reorder, drop, duplicate) are easy to detect by comparison.
func idsOfLen(n, size int) [][]byte {
	ids := make([][]byte, n)
	for i := range ids {
		id := make([]byte, size)
		binary.BigEndian.PutUint32(id[size-4:], uint32(i))
		ids[i] = id
	}
	return ids
}

func TestChunkTraceIDs_ExactChunkCount(t *testing.T) {
	const chunkSize = 4

	tests := []struct {
		name      string
		numIDs    int
		wantSizes []int
	}{
		{name: "empty", numIDs: 0, wantSizes: nil},
		{name: "single id", numIDs: 1, wantSizes: []int{1}},
		{name: "exactly one chunk", numIDs: chunkSize, wantSizes: []int{chunkSize}},
		{name: "one chunk plus one", numIDs: chunkSize + 1, wantSizes: []int{chunkSize, 1}},
		{name: "exactly three chunks", numIDs: 3 * chunkSize, wantSizes: []int{chunkSize, chunkSize, chunkSize}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := idsOfLen(tt.numIDs, 8)
			chunks := chunkTraceIDs(ids, chunkSize)

			require.Len(t, chunks, len(tt.wantSizes))
			for i, chunk := range chunks {
				assert.Len(t, chunk, tt.wantSizes[i])
			}
		})
	}
}

// TestChunkTraceIDs_IDPassthroughUnchanged locks in that chunking is purely
// a batching operation: every ID must survive byte-identical, in order,
// with no padding. Padding trace IDs to a canonical width is the reader/
// consumer's job (modules/bloomgateway/hash.go's
// util.PadTraceIDTo16Bytes call), not the producer's -- doing it here would
// make it impossible for a consumer to tell a genuinely-short ID from a
// padded one.
func TestChunkTraceIDs_IDPassthroughUnchanged(t *testing.T) {
	const chunkSize = 5

	for _, size := range []int{4, 8, 16} {
		t.Run(fmt.Sprintf("%d-byte ids", size), func(t *testing.T) {
			ids := idsOfLen(37, size) // deliberately not a multiple of chunkSize

			chunks := chunkTraceIDs(ids, chunkSize)

			var flattened [][]byte
			for _, chunk := range chunks {
				flattened = append(flattened, chunk...)
			}

			require.Equal(t, ids, flattened, "chunking must not add, drop, reorder, or duplicate any ID")
			for _, id := range flattened {
				assert.Len(t, id, size, "chunking must not pad IDs -- padding is consumer-side")
			}
		})
	}
}

func TestDedupeTraceIDs(t *testing.T) {
	t.Run("duplicates collapse and order is preserved", func(t *testing.T) {
		a, b, c := []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, []byte{9, 10, 11, 12}
		ids := [][]byte{a, b, a, c, b, a}

		assert.Equal(t, [][]byte{a, b, c}, dedupeTraceIDs(ids))
	})

	t.Run("empty input yields empty output", func(t *testing.T) {
		assert.Empty(t, dedupeTraceIDs(nil))
		assert.Empty(t, dedupeTraceIDs([][]byte{}))
	})

	t.Run("no duplicates leaves order untouched", func(t *testing.T) {
		ids := idsOfLen(10, 8)
		assert.Equal(t, ids, dedupeTraceIDs(ids))
	})
}

// TestPartitionForBlock_Stable exercises the two properties producers
// depend on: the partition key is a pure function of the block ID (so a
// republish of the same block deterministically lands in the same
// partition, DESIGN.md § Write path's "Partition key | block_id"), and the
// hash isn't so skewed that entire partitions starve.
func TestPartitionForBlock_Stable(t *testing.T) {
	t.Run("same block ID always routes to the same partition", func(t *testing.T) {
		id := backend.NewUUID()

		first := partitionForBlock(id, 16)
		second := partitionForBlock(id, 16)

		assert.Equal(t, first, second)
	})

	t.Run("loose distribution sanity: every bucket gets traffic", func(t *testing.T) {
		const (
			numPartitions = 16
			numBlocks     = 1000
		)
		counts := make([]int, numPartitions)

		for i := 0; i < numBlocks; i++ {
			p := partitionForBlock(backend.NewUUID(), numPartitions)
			counts[p]++
		}

		for p, count := range counts {
			assert.Positive(t, count, "partition %d received no blocks out of %d random UUIDs -- hash looks skewed", p, numBlocks)
		}
	})
}

func TestPartitionForBlock_InRange(t *testing.T) {
	const numPartitions = 16

	for i := 0; i < 1000; i++ {
		p := partitionForBlock(backend.NewUUID(), numPartitions)
		require.GreaterOrEqual(t, p, int32(0))
		require.Less(t, p, int32(numPartitions))
	}
}
