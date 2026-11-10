package tracesizes

import (
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metricTraceSizeBytes records the full size of a trace once it leaves the live
// set. Because the tracker keeps accumulating after a push is rejected for
// exceeding max_bytes_per_trace, the observation reflects the trace as
// received, not the truncated trace that ends up in storage. It reuses the
// metric name introduced for the distributor so existing queries and dashboards
// keep working, even though the observation now happens where the whole trace
// is available.
var metricTraceSizeBytes = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace:                       "tempo",
	Name:                            "distributor_trace_size_bytes",
	Help:                            "Total size in bytes of each trace, observed once when the trace is flushed, per tenant",
	NativeHistogramBucketFactor:     1.1,
	NativeHistogramMaxBucketNumber:  160,
	NativeHistogramMinResetDuration: 1 * time.Hour,
}, []string{"tenant"})

type Tracker struct {
	mtx      sync.Mutex
	hash     hash.Hash64
	sizes    map[uint64]*traceSize
	tenantID string
}

type traceSize struct {
	size      int
	timestamp time.Time
}

type AllowResult struct {
	IsAllowed        bool
	CurrentTotalSize int
}

func New(tenantID string) *Tracker {
	return &Tracker{
		hash:     fnv.New64(),
		sizes:    make(map[uint64]*traceSize),
		tenantID: tenantID,
	}
}

func (s *Tracker) token(traceID []byte) uint64 {
	s.hash.Reset()
	s.hash.Write(traceID)
	return s.hash.Sum64()
}

// Add adds sz to the tracked total for the trace and returns the new total. The
// total keeps growing even after the trace exceeds its size limit, so callers
// can observe how large a trace actually got.
func (s *Tracker) Add(traceID []byte, sz int) int {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.add(traceID, sz)
}

func (s *Tracker) add(traceID []byte, sz int) int {
	token := s.token(traceID)
	tr := s.sizes[token]
	if tr == nil {
		tr = &traceSize{
			size: 0, // size added below
		}
		s.sizes[token] = tr
	}

	tr.timestamp = time.Now()
	tr.size += sz

	return tr.size
}

// Allow returns true if the historical total plus incoming size is less than
// or equal to the max.  The historical total is kept alive and incremented even
// if not allowed, so that long-running traces are cutoff as expected.
func (s *Tracker) Allow(traceID []byte, sz, maxSize int) AllowResult {
	total := s.Add(traceID, sz)

	return AllowResult{
		total <= maxSize,
		total,
	}
}

func (s *Tracker) ClearIdle(idleSince time.Time) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for token, tr := range s.sizes {
		if tr.timestamp.Before(idleSince) {
			s.observe(tr.size)
			delete(s.sizes, token)
		}
	}
}

// Drain observes every tracked trace and forgets them. Call it once a block
// containing all live traces has been written so traces that never made it into
// the live set (every push exceeded max_bytes_per_trace) are still observed.
func (s *Tracker) Drain() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for token, tr := range s.sizes {
		s.observe(tr.size)
		delete(s.sizes, token)
	}
}

func (s *Tracker) observe(size int) {
	metricTraceSizeBytes.WithLabelValues(s.tenantID).Observe(float64(size))
}
