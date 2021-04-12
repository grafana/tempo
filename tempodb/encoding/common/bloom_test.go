package common

import (
	"bytes"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	willf_bloom "github.com/willf/bloom"
)

func TestShardedBloom(t *testing.T) {
	// create a bunch of traceIDs
	var err error
	const numTraces = 1000
	traceIDs := make([][]byte, 0)
	for i := 0; i < numTraces; i++ {
		id := make([]byte, 16)
		_, err = rand.Read(id)
		assert.NoError(t, err)
		traceIDs = append(traceIDs, id)
	}

	// create sharded bloom filter
	shardSize := 1000
	shardCount := 5
	estimatedObjects := 1000
	b := NewBloom(shardSize, shardCount, estimatedObjects)

	// add traceIDs to sharded bloom filter
	for _, traceID := range traceIDs {
		b.Add(traceID)
	}

	// get byte representation
	bloomBytes, err := b.Write()
	assert.NoError(t, err)
	assert.Len(t, bloomBytes, shardCount)

	// parse byte representation into willf_bloom.Bloomfilter
	var filters []*willf_bloom.BloomFilter
	for i := 0; i < shardCount; i++ {
		filters = append(filters, &willf_bloom.BloomFilter{})
	}
	for i, singleBloom := range bloomBytes {
		_, err = filters[i].ReadFrom(bytes.NewReader(singleBloom))
		assert.NoError(t, err)

		// assert that parsed form has the expected _m_ and _k_
		assert.Equal(t, filters[i].Cap(), uint(shardSize*8))                                       // * 8 because need bits from bytes
		assert.Equal(t, filters[i].K(), uint(evaluateK(shardSize*8, estimatedObjects/shardCount))) // * 8 because need bits from bytes
	}

	// confirm that the sharded bloom and parsed form give the same result
	missingCount := 0
	for _, traceID := range traceIDs {
		found := b.Test(traceID)
		if !found {
			missingCount++
		}
		assert.Equal(t, found, filters[ShardKeyForTraceID(traceID, shardCount)].Test(traceID))
	}

	// get estimated bloom filter false positive
	estimatedBloomFP := filters[0].EstimateFalsePositiveRate(uint(numTraces / shardCount))
	// check that missingCount is less than estimatedBloomFP
	assert.LessOrEqual(t, float64(missingCount), estimatedBloomFP*numTraces)
}

func TestEvaluateK(t *testing.T) {
	assert.Equal(t, int(math.Ceil(math.Ln2*float64(100))), evaluateK(1000, 10))
}
