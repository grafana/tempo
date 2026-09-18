package benchmark

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

// countingReader records the object-store traffic a query causes. It wraps the
// raw reader rather than reading counters off the response, so it sees the
// reads a block makes for footers, indexes and bloom filters too.
type countingReader struct {
	backend.RawReader

	reads  atomic.Int64
	bytes  atomic.Int64
	timeNs atomic.Int64
}

type readStats struct {
	Reads  int64 `json:"reads"`
	Bytes  int64 `json:"bytes"`
	TimeNs int64 `json:"timeNs"`
}

func (c *countingReader) snapshot() readStats {
	return readStats{Reads: c.reads.Load(), Bytes: c.bytes.Load(), TimeNs: c.timeNs.Load()}
}

func (c *countingReader) since(before readStats) readStats {
	now := c.snapshot()
	return readStats{
		Reads:  now.Reads - before.Reads,
		Bytes:  now.Bytes - before.Bytes,
		TimeNs: now.TimeNs - before.TimeNs,
	}
}

func (c *countingReader) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	start := time.Now()
	err := c.RawReader.ReadRange(ctx, name, keypath, offset, buffer, cacheInfo)
	c.record(int64(len(buffer)), time.Since(start))
	return err
}

func (c *countingReader) Read(ctx context.Context, name string, keyPath backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	start := time.Now()
	rc, size, err := c.RawReader.Read(ctx, name, keyPath, cacheInfo)
	c.record(size, time.Since(start))
	return rc, size, err
}

func (c *countingReader) record(n int64, took time.Duration) {
	c.reads.Add(1)
	if n > 0 {
		c.bytes.Add(n)
	}
	c.timeNs.Add(int64(took))
}
