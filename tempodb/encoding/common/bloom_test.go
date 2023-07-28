package common

import (
	"bytes"
	crand "crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	willf_bloom "github.com/willf/bloom"
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

	// parse byte representation into willf_bloom.Bloomfilter
	var filters []*willf_bloom.BloomFilter
	for i := 0; i < b.GetShardCount(); i++ {
		filters = append(filters, &willf_bloom.BloomFilter{})
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

			// parse byte representation into willf_bloom.Bloomfilter
			var filters []*willf_bloom.BloomFilter
			for i := 0; i < b.GetShardCount(); i++ {
				filters = append(filters, &willf_bloom.BloomFilter{})
			}

			for i, singleBloom := range bloomBytes {
				_, err = filters[i].ReadFrom(bytes.NewReader(singleBloom))
				assert.NoError(t, err)
				assert.LessOrEqual(t, filters[i].EstimateFalsePositiveRate(tt.estimatedObjects/uint(b.GetShardCount())), tt.bloomFP)
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
