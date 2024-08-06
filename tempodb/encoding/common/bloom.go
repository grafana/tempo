package common

import (
	"bytes"
	"math"

	"github.com/go-kit/log/level"
	"github.com/willf/bloom"

	"github.com/grafana/tempo/v2/pkg/util"
	"github.com/grafana/tempo/v2/pkg/util/log"
)

const (
	legacyShardCount = 10
	minShardCount    = 1
	maxShardCount    = 1000
)

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
	shardCount = uint(math.Ceil(float64(m) / (float64(shardSize) * 8.0)))

	if shardCount < minShardCount {
		shardCount = minShardCount
	}

	if shardCount > maxShardCount {
		shardCount = maxShardCount
		level.Warn(log.Logger).Log("msg", "required bloom filter shard count exceeded max. consider increasing bloom_filter_shard_size_bytes")
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
