package metrics

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

// CountingReader records the object-store traffic a query causes. It wraps the
// raw reader rather than reading the response, so it also sees the reads a
// block makes for footers, indexes and bloom filters.
type CountingReader struct {
	backend.RawReader

	reads  atomic.Int64
	bytes  atomic.Int64
	timeNs atomic.Int64
}

// ReadStats is the object-store traffic over one window.
type ReadStats struct {
	Reads  int64 `json:"reads"`
	Bytes  int64 `json:"bytes"`
	TimeNs int64 `json:"timeNs"`
}

func (c *CountingReader) Snapshot() ReadStats {
	return ReadStats{Reads: c.reads.Load(), Bytes: c.bytes.Load(), TimeNs: c.timeNs.Load()}
}

func (c *CountingReader) Since(before ReadStats) ReadStats {
	now := c.Snapshot()
	return ReadStats{
		Reads:  now.Reads - before.Reads,
		Bytes:  now.Bytes - before.Bytes,
		TimeNs: now.TimeNs - before.TimeNs,
	}
}

func (c *CountingReader) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	start := time.Now()
	err := c.RawReader.ReadRange(ctx, name, keypath, offset, buffer, cacheInfo)
	c.record(int64(len(buffer)), time.Since(start))
	return err
}

func (c *CountingReader) Read(ctx context.Context, name string, keyPath backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	start := time.Now()
	rc, size, err := c.RawReader.Read(ctx, name, keyPath, cacheInfo)
	c.record(size, time.Since(start))
	return rc, size, err
}

func (c *CountingReader) record(n int64, took time.Duration) {
	c.reads.Add(1)
	if n > 0 {
		c.bytes.Add(n)
	}
	c.timeNs.Add(int64(took))
}
