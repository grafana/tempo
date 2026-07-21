package bloomgatewayevents

import (
	"github.com/cespare/xxhash/v2"

	"github.com/grafana/tempo/tempodb/backend"
)

// dedupeTraceIDs returns ids with exact byte-for-byte duplicates removed,
// keeping the first occurrence's position. Order-preserving so callers that
// care about original arrival order (e.g. for reproducible chunk contents)
// aren't surprised by a reshuffle.
func dedupeTraceIDs(ids [][]byte) [][]byte {
	seen := make(map[string]struct{}, len(ids))
	out := make([][]byte, 0, len(ids))
	for _, id := range ids {
		key := string(id) // string(byteSlice) copies the bytes into the map key; out still holds the original slice
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}
	return out
}

// chunkTraceIDs splits ids into consecutive slices of at most chunkSize
// elements each (the last chunk may be smaller), preserving order and
// without copying or padding any ID -- padding to a canonical width is the
// consumer's job (DESIGN.md § Design constraints: exactly one padding call
// site, upstream of every computed hash). Chunks are sub-slices of ids
// rather than copies, so callers must treat the input as shared once
// chunked, the same as any other slicing operation.
func chunkTraceIDs(ids [][]byte, chunkSize int) [][][]byte {
	if len(ids) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		// Config.Validate is expected to already guarantee ChunkSize > 0
		// upstream of every real caller; this only prevents a defensive
		// caller bug from turning into an infinite loop below.
		chunkSize = len(ids)
	}

	chunks := make([][][]byte, 0, (len(ids)+chunkSize-1)/chunkSize)
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}

// partitionForBlock returns the Kafka partition a block's events should be
// published to: xxhash64 of the block ID's 16 raw bytes, reduced mod
// numPartitions. A pure function of blockID so republishing the same block
// (e.g. after a producer retry) always lands on the same partition
// (DESIGN.md § Write path: "Partition key | block_id"). numPartitions <= 0
// is a caller bug (there is no valid partition to return); rather than
// panic on a mod-by-zero, it degrades to partition 0.
func partitionForBlock(blockID backend.UUID, numPartitions int32) int32 {
	if numPartitions <= 0 {
		return 0
	}
	sum := xxhash.Sum64(blockID[:])
	return int32(sum % uint64(numPartitions))
}
