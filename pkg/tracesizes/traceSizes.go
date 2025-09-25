package tracesizes

import (
	"hash"
	"hash/fnv"
	"sync"
	"time"
)

type Tracker struct {
	mtx   sync.Mutex
	hash  hash.Hash64
	sizes map[uint64]*traceSize
}

type traceSize struct {
	size      int
	timestamp time.Time
}

type AllowResult struct {
	IsAllowed        bool
	CurrentTotalSize int
}

func New() *Tracker {
	return &Tracker{
		hash:  fnv.New64(),
		sizes: make(map[uint64]*traceSize),
	}
}

func (s *Tracker) token(traceID []byte) uint64 {
	s.hash.Reset()
	s.hash.Write(traceID)
	return s.hash.Sum64()
}

// Allow returns true if the historical total plus incoming size is less than
// or equal to the max.  The historical total is kept alive and incremented even
// if not allowed, so that long-running traces are cutoff as expected.
func (s *Tracker) Allow(traceID []byte, sz, maxSize int) AllowResult {
	s.mtx.Lock()
	defer s.mtx.Unlock()

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

	return AllowResult{
		tr.size <= maxSize,
		tr.size,
	}
}

func (s *Tracker) ClearIdle(idleSince time.Time) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for token, tr := range s.sizes {
		if tr.timestamp.Before(idleSince) {
			delete(s.sizes, token)
		}
	}
}
