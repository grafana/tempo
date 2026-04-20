package common

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardedBloom(t *testing.T) {
	// create a bunch of traceIDs
	var err error
	const numTraces = 10000
	traceIDs := make([][]byte, 0)
	for i := 0; i < numTraces; i++ {
		id := make([]byte, 16)
		_, err = crand.Read(id)
		assert.NoError(t, err)
		traceIDs = append(traceIDs, id)
	}

	// create sharded bloom filter
	const bloomFP = .01
	shardSize := uint(100)
	estimatedObjects := uint(numTraces)
	b := NewBloom(bloomFP, shardSize, estimatedObjects)

	// add traceIDs to sharded bloom filter
	for _, traceID := range traceIDs {
		b.Add(traceID)
	}

	// get byte representation
	bloomBytes, err := b.Marshal()
	assert.NoError(t, err)
	assert.Len(t, bloomBytes, b.GetShardCount())

	// parse byte representation into bloom.Bloomfilter
	var filters []*bloom.BloomFilter
	for i := 0; i < b.GetShardCount(); i++ {
		filters = append(filters, &bloom.BloomFilter{})
	}
	for i, singleBloom := range bloomBytes {
		_, err = filters[i].ReadFrom(bytes.NewReader(singleBloom))
		assert.NoError(t, err)

		// assert that parsed form has the expected size
		assert.Equal(t, shardSize*8, filters[i].Cap()) // * 8 because need bits from bytes
	}

	// confirm that the sharded bloom and parsed form give the same result
	missingCount := 0
	for _, traceID := range traceIDs {
		found := b.Test(traceID)
		if !found {
			missingCount++
		}
		assert.Equal(t, found, filters[ShardKeyForTraceID(traceID, b.GetShardCount())].Test(traceID))
	}

	// check that missingCount is less than bloomFP
	assert.LessOrEqual(t, float64(missingCount), bloomFP*numTraces)
}

func TestShardedBloomFalsePositive(t *testing.T) {
	tests := []struct {
		name             string
		bloomFP          float64
		shardSize        uint
		estimatedObjects uint
	}{
		{
			name:             "regular",
			bloomFP:          0.05,
			shardSize:        250 * 1024,
			estimatedObjects: 10_000_000,
		},
		{
			name:             "large estimated objects",
			bloomFP:          0.01,
			shardSize:        100,
			estimatedObjects: 10000,
		},
		{
			name:             "large shard size",
			bloomFP:          0.01,
			shardSize:        100000,
			estimatedObjects: 10,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable, needed for running test cases in parallel
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := NewBloom(tt.bloomFP, tt.shardSize, tt.estimatedObjects)

			// get byte representation
			bloomBytes, err := b.Marshal()
			assert.NoError(t, err)

			// parse byte representation into bloom.Bloomfilter
			var filters []*bloom.BloomFilter
			for i := 0; i < b.GetShardCount(); i++ {
				filters = append(filters, &bloom.BloomFilter{})
			}

			for i, singleBloom := range bloomBytes {
				_, err = filters[i].ReadFrom(bytes.NewReader(singleBloom))
				assert.NoError(t, err)
				assert.LessOrEqual(t, bloom.EstimateFalsePositiveRate(filters[i].Cap(), filters[i].K(), tt.estimatedObjects/uint(b.GetShardCount())), tt.bloomFP)
			}
		})
	}
}

func TestBloomShardCount(t *testing.T) {
	tests := []struct {
		name             string
		bloomFP          float64
		shardSize        uint
		estimatedObjects uint
		expectedShards   uint
	}{
		{
			name:             "too many shards",
			bloomFP:          0.01,
			shardSize:        1,
			estimatedObjects: 100000,
			expectedShards:   maxShardCount,
		},
		{
			name:             "too few shards",
			bloomFP:          0.01,
			shardSize:        10,
			estimatedObjects: 1,
			expectedShards:   minShardCount,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable, needed for running test cases in parallel
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := NewBloom(tt.bloomFP, tt.shardSize, tt.estimatedObjects)
			assert.Equal(t, int(tt.expectedShards), b.GetShardCount())
		})
	}
}

// test bloom params used to create the test bloom blobs.
const (
	testBloomBlobFP               = 0.01
	testBloomBlobShardSizeBytes   = 20
	testBloomBlobEstimatedObjects = 50
)

// testBloomBlobPresentIDs are the trace IDs that were added when the test bloom blobs were generated.
// Every one of these must be found!
var testBloomBlobPresentIDs = [][]byte{
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a},
}

// TestBloomFilterCompatibility verifies that bloom filter bytes produced by github.com/willf/bloom (stored in ./test-data)
// current bloom filter library
func TestBloomFilterCompatibility(t *testing.T) {
	b := NewBloom(testBloomBlobFP, testBloomBlobShardSizeBytes, testBloomBlobEstimatedObjects)
	for _, id := range testBloomBlobPresentIDs {
		b.Add(id)
	}
	shardCount := b.GetShardCount()

	// Uncomment to regenerate the bloom blobs. Do this only when the bloom filter implementation changes!
	// regenerateBloomBlobs(t, b)

	for i := 0; i < shardCount; i++ {
		blobPath := fmt.Sprintf("test-data/bloom-%d", i)
		blobBytes, err := os.ReadFile(blobPath)
		require.NoError(t, err, "missing test blob %s, uncomment regenerateBloomBlobs() to re-generate files", blobPath)

		filter := &bloom.BloomFilter{}
		_, err = filter.ReadFrom(bytes.NewReader(blobBytes))
		require.NoError(t, err, "failed to deserialize bloom shard %d", i)

		for _, id := range testBloomBlobPresentIDs {
			if ShardKeyForTraceID(id, shardCount) == i {
				assert.True(t, filter.Test(id), "shard %d: trace ID %x missing from deserialized bloom filter", i, id)
			}
		}
	}
}

// nolint:unused
func regenerateBloomBlobs(t *testing.T, b *ShardedBloomFilter) {
	t.Helper()
	require.NoError(t, os.MkdirAll("test-data", 0o755))
	bloomBytes, err := b.Marshal()
	require.NoError(t, err)
	for i, shard := range bloomBytes {
		path := fmt.Sprintf("test-data/bloom-%d", i)
		require.NoError(t, os.WriteFile(path, shard, 0o644))
	}
	t.Logf("wrote %d test bloom blobs to test-data/ - commit them", len(bloomBytes))
}
