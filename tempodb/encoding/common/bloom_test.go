package common

import (
	"bytes"
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
	const bloomFP = .01
	b := NewWithEstimates(uint(numTraces), bloomFP)

	// add traceIDs to sharded bloom filter
	for _, traceID := range traceIDs {
		b.Add(traceID)
	}

	// get byte representation
	bloomBytes, err := b.WriteTo()
	assert.NoError(t, err)
	assert.Len(t, bloomBytes, shardNum)

	// parse byte representation into willf_bloom.Bloomfilter
	var filters []*willf_bloom.BloomFilter
	for i := 0; i < shardNum; i++ {
		filters = append(filters, &willf_bloom.BloomFilter{})
	}
	for i, singleBloom := range bloomBytes {
		_, err = filters[i].ReadFrom(bytes.NewReader(singleBloom))
		assert.NoError(t, err)
	}

	// confirm that the sharded bloom and parsed form give the same result
	missingCount := 0
	for _, traceID := range traceIDs {
		found := b.Test(traceID)
		if !found {
			missingCount++
		}
		assert.Equal(t, found, filters[ShardKeyForTraceID(traceID)].Test(traceID))
	}

	// check that missingCount is less than bloomFP
	assert.LessOrEqual(t, float64(missingCount), bloomFP*numTraces)
}
