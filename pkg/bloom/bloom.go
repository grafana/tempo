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
		bloomBuffer := &bytes.Buffer{}
		_, err := f.WriteTo(bloomBuffer)
		if err != nil {
			return nil, err
		}
		bloomBytes[i] = bloomBuffer.Bytes()
	}
	return bloomBytes, nil
}

func ShardKeyForTraceID(traceID []byte) uint64 {
	return util.Fingerprint(traceID) % shardNum
}

func GetShardNum() int {
	return shardNum
}