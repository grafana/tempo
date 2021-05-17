package common

import (
	"bytes"
	"math"

	"github.com/willf/bloom"

	"github.com/grafana/tempo/pkg/util"
)

const legacyShardCount = 10

type ShardedBloomFilter struct {
	blooms []*bloom.BloomFilter
}

func evaluateK(shardSizeInBits, itemsPerBloom int) (k int) {
	// Per https://llimllib.github.io/bloomfilter-tutorial/ under "How many hash functions should I use?"
	// the optimal value of k: (m/n)ln(2)
	// m: number of bits in the filter
	// n: estimated number of objects
	// k: number of hash functions
	k = int(math.Ceil((float64(shardSizeInBits) / float64(itemsPerBloom)) * (math.Ln2)))

	return
}

// NewBloom creates a ShardedBloomFilter
func NewBloom(fp float64, shardSize, estimatedObjects uint) *ShardedBloomFilter {
	// estimate the number of shards needed. an approximate value is enough
	var shardCount uint
	var kPerBloom uint
	for {
		shardCount++
		var m, k uint
		if m, k = bloom.EstimateParameters(estimatedObjects/shardCount, fp); m < shardSize {
			kPerBloom = k
			break
		}
	}

	b := &ShardedBloomFilter{
		blooms: make([]*bloom.BloomFilter, shardCount),
	}

	for i := 0; i < int(shardCount); i++ {
		// New(m uint, k uint) creates a new Bloom filter with _m_ bits and _k_ hashing functions
		b.blooms[i] = bloom.New(shardSize*8, kPerBloom)
	}

	return b
}

func (b *ShardedBloomFilter) Add(traceID []byte) {
	shardKey := ShardKeyForTraceID(traceID, len(b.blooms))
	b.blooms[shardKey].Add(traceID)
}

// Write is a wrapper around bloom.WriteTo
func (b *ShardedBloomFilter) Write() ([][]byte, error) {
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
