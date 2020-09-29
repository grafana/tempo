package bloom

import (
	"github.com/willf/bloom"
	"io"

	"github.com/grafana/tempo/pkg/util"
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

// TODO
func (f *ShardedBloomFilter) WriteTo(stream io.Writer) (int64, error) {
	return 0, nil
}

// TODO
func (f *ShardedBloomFilter) Test(traceID []byte) bool {
	return true
}
