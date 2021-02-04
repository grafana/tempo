package encoding

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type CompactorBlock struct {
	encoding versionedEncoding

	compactedMeta *backend.BlockMeta
	inMetas       []*backend.BlockMeta

	bloom *common.ShardedBloomFilter

	bufferedObjects int
	appendBuffer    *bytes.Buffer
	appender        common.Appender
}

// NewCompactorBlock creates a ... new compactor block!
func NewCompactorBlock(cfg *BlockConfig, id uuid.UUID, tenantID string, metas []*backend.BlockMeta, estimatedObjects int) (*CompactorBlock, error) {
	if len(metas) == 0 {
		return nil, fmt.Errorf("empty block meta list")
	}

	if estimatedObjects <= 0 {
		return nil, fmt.Errorf("must have non-zero positive estimated objects for a reliable bloom filter")
	}

	c := &CompactorBlock{
		encoding:      latestEncoding(),
		compactedMeta: backend.NewBlockMeta(tenantID, id, currentVersion, cfg.Encoding),
		bloom:         common.NewWithEstimates(uint(estimatedObjects), cfg.BloomFP),
		inMetas:       metas,
	}

	var err error
	c.appendBuffer = &bytes.Buffer{}
	c.appender, err = c.encoding.newBufferedAppender(c.appendBuffer, cfg.Encoding, cfg.IndexDownsample, estimatedObjects)
	if err != nil {
		return nil, fmt.Errorf("failed to created appender: %w", err)
	}

	return c, nil
}

func (c *CompactorBlock) AddObject(id common.ID, object []byte) error {
	err := c.appender.Append(id, object)
	if err != nil {
		return err
	}
	c.bufferedObjects++
	c.compactedMeta.ObjectAdded(id)
	c.bloom.Add(id)
	return nil
}

func (c *CompactorBlock) CurrentBufferLength() int {
	return c.appendBuffer.Len()
}

func (c *CompactorBlock) CurrentBufferedObjects() int {
	return c.bufferedObjects
}

func (c *CompactorBlock) Length() int {
	return c.appender.Length()
}

// FlushBuffer flushes any existing objects to the backend
func (c *CompactorBlock) FlushBuffer(ctx context.Context, tracker backend.AppendTracker, w backend.Writer) (backend.AppendTracker, int, error) {
	if c.appender.Length() == 0 {
		return tracker, 0, nil
	}

	meta := c.BlockMeta()
	tracker, err := c.encoding.appendBlockData(ctx, w, meta, tracker, c.appendBuffer.Bytes())
	if err != nil {
		return nil, 0, err
	}

	bytesFlushed := c.appendBuffer.Len()
	c.appendBuffer.Reset()
	c.bufferedObjects = 0

	return tracker, bytesFlushed, nil
}

// Complete finishes writes the compactor metadata and closes all buffers and appenders
func (c *CompactorBlock) Complete(ctx context.Context, tracker backend.AppendTracker, w backend.Writer) (int, error) {
	err := c.appender.Complete()
	if err != nil {
		return 0, err
	}

	// one final flush
	tracker, bytesFlushed, err := c.FlushBuffer(ctx, tracker, w)
	if err != nil {
		return 0, err
	}

	records := c.appender.Records()
	meta := c.BlockMeta()

	err = c.encoding.writeBlockMeta(ctx, w, meta, records, c.bloom)
	if err != nil {
		return 0, err
	}

	return bytesFlushed, w.CloseAppend(ctx, tracker)
}

func (c *CompactorBlock) BlockMeta() *backend.BlockMeta {
	meta := c.compactedMeta

	meta.StartTime = c.inMetas[0].StartTime
	meta.EndTime = c.inMetas[0].EndTime

	// everything should be correct here except the start/end times which we will get from the passed in metas
	for _, m := range c.inMetas[1:] {
		if m.StartTime.Before(meta.StartTime) {
			meta.StartTime = m.StartTime
		}
		if m.EndTime.After(meta.EndTime) {
			meta.EndTime = m.EndTime
		}
	}

	return meta
}
