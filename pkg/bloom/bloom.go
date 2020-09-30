package bloom

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
	for i := 0; i < shardNum; i ++ {
		b.blooms[i] = bloom.NewWithEstimates(n/shardNum, fp)
	}

	return b
}

func (b *ShardedBloomFilter) Add(traceID []byte) {
	shardKey := util.Fingerprint(traceID) % shardNum
	b.blooms[shardKey].Add(traceID)
}

// WriteTo is a wrapper around bloom.WriteTo
func (b *ShardedBloomFilter) WriteTo() ([][]byte, error) {
	bloomBytes := make([][]byte, 10)
	for i, f := range b.blooms {
		_, err := f.WriteTo(bytes.NewBuffer(bloomBytes[i]))
		if err != nil {
			return nil, err
		}
	}
	return bloomBytes, nil
}

// Test implements bloom.Test
func (b *ShardedBloomFilter) Test(traceID []byte) bool {
	shardKey := util.Fingerprint(traceID) % shardNum
	return b.blooms[shardKey].Test(traceID)
}

func ShardKeyForTraceID(traceID []byte) uint64 {
	return util.Fingerprint(traceID) % shardNum
}

func GetShardNum() int {
	return shardNum
}