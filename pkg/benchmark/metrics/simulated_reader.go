package metrics

import (
	"context"
	"io"
	"time"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

// SimulatedReader makes a local block read like one in an object store. Each
// request costs a fixed latency plus its transfer time at a per-request
// bandwidth, which is the trade-off the read buffer size moves: fewer, larger
// requests against more bytes read that a query never needed.
//
// The cost is slept rather than added up, so requests the parquet reader
// issues concurrently overlap as they would against a real store.
type SimulatedReader struct {
	backend.RawReader

	Latency time.Duration
	// Bandwidth is in bytes per second. Zero is unlimited.
	Bandwidth int64

	sleep func(context.Context, time.Duration) error
}

// NewSimulatedReader returns r unchanged when there is nothing to simulate.
func NewSimulatedReader(r backend.RawReader, latency time.Duration, bandwidth int64) backend.RawReader {
	if latency <= 0 && bandwidth <= 0 {
		return r
	}
	return &SimulatedReader{RawReader: r, Latency: latency, Bandwidth: bandwidth, sleep: sleepContext}
}

func (s *SimulatedReader) ReadRange(ctx context.Context, name string, keypath backend.KeyPath, offset uint64, buffer []byte, cacheInfo *backend.CacheInfo) error {
	if err := s.RawReader.ReadRange(ctx, name, keypath, offset, buffer, cacheInfo); err != nil {
		return err
	}
	return s.sleep(ctx, s.cost(int64(len(buffer))))
}

func (s *SimulatedReader) Read(ctx context.Context, name string, keyPath backend.KeyPath, cacheInfo *backend.CacheInfo) (io.ReadCloser, int64, error) {
	rc, size, err := s.RawReader.Read(ctx, name, keyPath, cacheInfo)
	if err != nil {
		return nil, 0, err
	}
	if err := s.sleep(ctx, s.cost(size)); err != nil {
		rc.Close()
		return nil, 0, err
	}
	return rc, size, nil
}

func (s *SimulatedReader) cost(n int64) time.Duration {
	d := max(s.Latency, 0)
	if s.Bandwidth > 0 && n > 0 {
		d += time.Duration(float64(n) / float64(s.Bandwidth) * float64(time.Second))
	}
	return d
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
