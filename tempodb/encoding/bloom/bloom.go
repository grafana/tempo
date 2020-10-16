package bloom

import (
	"bytes"

	"github.com/willf/bloom"
)

const shardNum = 16

type ShardedBloomFilter struct {
	blooms []*bloom.BloomFilter
}

func NewWithEstimates(n uint, fp float64) *ShardedBloomFilter {
	b := &ShardedBloomFilter{
		blooms: make([]*bloom.BloomFilter, shardNum),
	}
	for i := 0; i < shardNum; i++ {
		b.blooms[i] = bloom.NewWithEstimates(n/shardNum, fp)
	}

	return b
}

func (b *ShardedBloomFilter) Add(traceID []byte) {
	shardKey := ShardKeyForTraceID(traceID)
	b.blooms[shardKey].Add(traceID)
}

// WriteTo is a wrapper around bloom.WriteTo
func (b *ShardedBloomFilter) WriteTo() ([][]byte, error) {
	bloomBytes := make([][]byte, shardNum)
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

func ShardKeyForTraceID(traceID []byte) int {
	// least significant bits obtained by &'ing the last byte with a 0x0F
	return int(traceID[len(traceID)-1] & 0x0F)
}

// Test implements bloom.Test -> required only for testing
func (b *ShardedBloomFilter) Test(traceID []byte) bool {
	shardKey := ShardKeyForTraceID(traceID)
	return b.blooms[shardKey].Test(traceID)
}

func GetShardNum() int {
	return shardNum
}
