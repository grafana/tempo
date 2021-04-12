package common

import (
	"bytes"
	"math"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/willf/bloom"

	"github.com/grafana/tempo/pkg/util"
)

const legacyShardCount = 10

var (
	metricBloomFP = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "bloom_filter_false_positive",
		Help:      "False positive values for bloom filters created",
		// 0.005, 0.020, 0.080, 0.32
		Buckets: prometheus.ExponentialBuckets(0.005, 4, 4),
	})
)

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
func NewBloom(shardSize, shardCount, estimatedObjects int) *ShardedBloomFilter {
	itemsPerBloom := estimatedObjects / shardCount
	if itemsPerBloom == 0 {
		itemsPerBloom = 1
	}

	shardSizeInBits := 8 * shardSize
	k := evaluateK(shardSizeInBits, itemsPerBloom)

	b := &ShardedBloomFilter{
		blooms: make([]*bloom.BloomFilter, shardCount),
	}
	for i := 0; i < shardCount; i++ {
		// New(m uint, k uint) creates a new Bloom filter with _m_ bits and _k_ hashing functions
		b.blooms[i] = bloom.New(uint(shardSizeInBits), uint(k))
	}

	// metric the false positive rate so we can track if we're making bad blooms
	metricBloomFP.Observe(b.blooms[0].EstimateFalsePositiveRate(uint(itemsPerBloom)))

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
