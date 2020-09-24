package wal

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/willf/bloom"
)

type CompactorBlock struct {
	block

	metas []*encoding.BlockMeta

	bloom *bloom.BloomFilter

	appendBuffer *bytes.Buffer
	appender     encoding.Appender
}

func newCompactorBlock(id uuid.UUID, tenantID string, bloomFP float64, indexDownsample int, metas []*encoding.BlockMeta, filepath string, estimatedObjects int) (*CompactorBlock, error) {
	if len(metas) == 0 {
		return nil, fmt.Errorf("empty block meta list")
	}

	if estimatedObjects <= 0 {
		return nil, fmt.Errorf("must have non-zero positive estimated objects for a reliable bloom filter")
	}

	c := &CompactorBlock{
		block: block{
			meta:     encoding.NewBlockMeta(tenantID, id),
			filepath: filepath,
		},
		bloom: bloom.NewWithEstimates(uint(estimatedObjects), bloomFP),
		metas: metas,
	}

	name := c.fullFilename()
	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	c.appendBuffer = &bytes.Buffer{}
	c.appender = encoding.NewBufferedAppender(c.appendBuffer, indexDownsample, estimatedObjects)

	return c, nil
}

func (c *CompactorBlock) Write(id encoding.ID, object []byte) error {
	err := c.appender.Append(id, object)
	if err != nil {
		return err
	}
	c.meta.ObjectAdded(id)
	c.bloom.Add(id)
	return nil
}

func (c *CompactorBlock) CurrentBuffer() []byte {
	return c.appendBuffer.Bytes()
}

func (c *CompactorBlock) ResetBuffer() {
	c.appendBuffer.Reset()
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
func (c *CompactorBlock) BlockMeta() *encoding.BlockMeta {
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
func (c *CompactorBlock) BloomFilter() *bloom.BloomFilter {
	return c.bloom
}

// implements WriteableBlock
func (c *CompactorBlock) Flushed(t time.Time) {
	// no-op
}

// implements WriteableBlock
func (c *CompactorBlock) Records() []*encoding.Record {
	return c.appender.Records()
}

// implements WriteableBlock
func (c *CompactorBlock) ObjectFilePath() string {
	return ""
}
