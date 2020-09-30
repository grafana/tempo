package wal

import (
	"fmt"
	"os"
	"sync"

	"github.com/grafana/tempo/pkg/bloom"
	"github.com/grafana/tempo/tempodb/encoding"
)

type WriteableBlock interface {
	BlockMeta() *encoding.BlockMeta
	BloomFilter() *bloom.ShardedBloomFilter
	Records() []*encoding.Record
	ObjectFilePath() string

	Flushed() error
}

type block struct {
	meta     *encoding.BlockMeta
	filepath string
	readFile *os.File

	once sync.Once
}

func (b *block) fullFilename() string {
	return fmt.Sprintf("%s/%v:%v", b.filepath, b.meta.BlockID, b.meta.TenantID)
}

func (b *block) file() (*os.File, error) {
	var err error
	b.once.Do(func() {
		if b.readFile == nil {
			name := b.fullFilename()

			b.readFile, err = os.OpenFile(name, os.O_RDONLY, 0644)
		}
	})

	return b.readFile, err
}
