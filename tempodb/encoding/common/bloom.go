package common

import (
	"bytes"

	"github.com/grafana/tempo/pkg/util"
	"github.com/willf/bloom"
)

const shardNum = 10

type ShardedBloomFilter struct {
	blooms []*bloom.BloomFilter
}

func NewWithEstimates(n uint, fp float64) *ShardedBloomFilter {
	b := &ShardedBloomFilter{
		blooms: make([]*bloom.BloomFilter, shardNum),
	}

	itemsPerBloom := n / shardNum
	if itemsPerBloom == 0 {
		itemsPerBloom = 1
	}
	for i := 0; i < shardNum; i++ {
		b.blooms[i] = bloom.NewWithEstimates(itemsPerBloom, fp)
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
	return int(util.TokenForTraceID(traceID)) % shardNum
}

// Test implements bloom.Test -> required only for testing
func (b *ShardedBloomFilter) Test(traceID []byte) bool {
	shardKey := ShardKeyForTraceID(traceID)
	return b.blooms[shardKey].Test(traceID)
}

func GetShardNum() int {
	return shardNum
}
