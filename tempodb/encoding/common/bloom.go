package common

import (
	"bytes"

	"github.com/willf/bloom"

	"github.com/grafana/tempo/pkg/util"
)

const legacyShardCount = 10

type ShardedBloomFilter struct {
	blooms []*bloom.BloomFilter
}

// NewBloom creates a ShardedBloomFilter
func NewBloom(fp float64, shardSize, estimatedObjects uint) *ShardedBloomFilter {
	// estimate the number of shards needed
	// m: number of bits in the filter
	// k: number of hash functions
	var shardCount uint
	m, k := bloom.EstimateParameters(estimatedObjects, fp)
	if m%(shardSize*8) == 0 {
		shardCount = m / (shardSize * 8)
	} else {
		shardCount = (m / (shardSize * 8)) + 1
	}

	b := &ShardedBloomFilter{
		blooms: make([]*bloom.BloomFilter, shardCount),
	}

	for i := 0; i < int(shardCount); i++ {
		// New(m uint, k uint) creates a new Bloom filter with _m_ bits and _k_ hashing functions
		b.blooms[i] = bloom.New(shardSize*8, k)
	}

	return b
}

func (b *ShardedBloomFilter) Add(traceID []byte) {
	shardKey := ShardKeyForTraceID(traceID, len(b.blooms))
	b.blooms[shardKey].Add(traceID)
}

// Marshal is a wrapper around bloom.WriteTo
func (b *ShardedBloomFilter) Marshal() ([][]byte, error) {
	bloomBytes := make([][]byte, len(b.blooms))
	for i, f := range b.blooms {
		bloomBuffer := &bytes.Buffer{}
		_, err := f.WriteTo(bloomBuffer)
		if err != nil {
			return nil, err
		}
		bloomBytes[i] = bloomBuffer.Bytes()
	}
	return bloomBytes, nil
}

func (b *ShardedBloomFilter) GetShardCount() int {
	return len(b.blooms)
}

// Test implements bloom.Test -> required only for testing
func (b *ShardedBloomFilter) Test(traceID []byte) bool {
	shardKey := ShardKeyForTraceID(traceID, len(b.blooms))
	return b.blooms[shardKey].Test(traceID)
}

func ShardKeyForTraceID(traceID []byte, shardCount int) int {
	return int(util.TokenForTraceID(traceID)) % ValidateShardCount(shardCount)
}

// For backward compatibility
func ValidateShardCount(shardCount int) int {
	if shardCount == 0 {
		return legacyShardCount
	}
	return shardCount
}
