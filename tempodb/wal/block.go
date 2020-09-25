package wal

import (
	"fmt"
	"os"
	"time"

	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/willf/bloom"
)

type WriteableBlock interface {
	BlockMeta() *encoding.BlockMeta
	BloomFilter() *bloom.BloomFilter
	Records() []*encoding.Record
	ObjectFilePath() string

	Flushed(flushTime time.Time) error
}

type block struct {
	meta     *encoding.BlockMeta
	filepath string
	readFile *os.File
}

func (b *block) fullFilename() string {
	return fmt.Sprintf("%s/%v:%v", b.filepath, b.meta.BlockID, b.meta.TenantID)
}

func (b *block) file() (*os.File, error) {
	if b.readFile == nil {
		name := b.fullFilename()

		f, err := os.OpenFile(name, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		b.readFile = f
	}

	return b.readFile, nil
}
