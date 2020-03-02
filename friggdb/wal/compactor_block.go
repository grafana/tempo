package wal

import (
	"fmt"
	"os"
	"time"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/dgryski/go-farm"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
)

// compactor block wraps a headblock to facilitate compaction.  it primarily needs the headblock
//   append code and then provides helper methods to massage the
// feel like this is kind of not worth it.  if tests start failing probably a good idea to just
//   split this functionality entirely
type CompactorBlock struct {
	block

	metas []*backend.BlockMeta

	bloom      *bloom.Bloom
	appendFile *os.File //jpe : consolidate in block?
	appender   backend.Appender
}

func newCompactorBlock(id uuid.UUID, tenantID string, bloomFP float64, indexDownsample int, metas []*backend.BlockMeta, filepath string, estimatedObjects int) (*CompactorBlock, error) {
	if len(metas) == 0 {
		return nil, fmt.Errorf("empty block meta list")
	}

	if estimatedObjects <= 0 {
		return nil, fmt.Errorf("must have non-zero positive estimated objects for a reliable bloom filter")
	}

	c := &CompactorBlock{
		block: block{
			meta:     backend.NewBlockMeta(tenantID, id),
			filepath: filepath,
		},
		bloom: bloom.NewBloomFilter(float64(estimatedObjects), bloomFP),
		metas: metas,
	}

	name := c.fullFilename()
	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	c.appendFile = f
	c.appender = backend.NewBufferedAppender(c.appendFile, indexDownsample, estimatedObjects)

	return c, nil
}

func (c *CompactorBlock) Write(id backend.ID, object []byte) error {
	err := c.appender.Append(id, object)
	if err != nil {
		return err
	}
	c.meta.ObjectAdded(id)
	c.bloom.Add(farm.Fingerprint64(id))
	return nil
}

func (c *CompactorBlock) Length() int {
	return c.appender.Length()
}

func (c *CompactorBlock) Complete() {
	c.appender.Complete()
}

func (c *CompactorBlock) Clear() error {
	return os.Remove(c.fullFilename())
}

// implements WriteableBlock
func (c *CompactorBlock) BlockMeta() *backend.BlockMeta {
	meta := c.meta

	meta.StartTime = c.metas[0].StartTime
	meta.EndTime = c.metas[0].EndTime

	// everything should be correct here except the start/end times which we will get from the passed in metas
	for _, m := range c.metas[1:] {
		if m.StartTime.Before(meta.StartTime) {
			meta.StartTime = m.StartTime
		}
		if m.EndTime.After(meta.EndTime) {
			meta.EndTime = m.EndTime
		}
	}

	return meta
}

// implements WriteableBlock
func (c *CompactorBlock) BloomFilter() *bloom.Bloom {
	return c.bloom
}

// implements WriteableBlock
func (c *CompactorBlock) BlockWroteSuccessfully(t time.Time) {
	// no-op
}

// implements WriteableBlock
func (c *CompactorBlock) Records() []*backend.Record {
	return c.appender.Records()
}

func (c *CompactorBlock) ObjectFilePath() string {
	return c.fullFilename()
}
