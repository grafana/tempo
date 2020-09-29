package bloom

import (
	"bytes"
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

func (b *ShardedBloomFilter) ReadFrom(bloomBytes [][]byte) (int64, error) {
	for i, bb := range bloomBytes {
		b.blooms[i].ReadFrom(bytes.NewReader(bb))
	}
	bloom.BloomFilter{}.ReadFrom()
}

// TODO
func (f *ShardedBloomFilter) Test(traceID []byte) bool {
	return true
}
