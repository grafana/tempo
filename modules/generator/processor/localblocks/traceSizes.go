package localblocks

import (
	"hash"
	"hash/fnv"
	"sync"
	"time"
)

type traceSizes struct {
	mtx   sync.Mutex
	hash  hash.Hash64
	sizes map[uint64]*traceSize
}

type traceSize struct {
	size      int
	timestamp time.Time
}

func newTraceSizes() *traceSizes {
	return &traceSizes{
		hash:  fnv.New64(),
		sizes: map[uint64]*traceSize{},
	}
}

func (s *traceSizes) token(traceID []byte) uint64 {
	s.hash.Reset()
	s.hash.Write(traceID)
	return s.hash.Sum64()
}

// Allow returns true if the total previously received plus incoming sizes are less than
// or equal to the max.  The incoming data size is added to the historical total even
// if not allowed, so that when the max limit changes it still evalulates as expected.
func (s *traceSizes) Allow(traceID []byte, sz, max int) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	token := s.token(traceID)
	tr := s.sizes[token]
	if tr == nil {
		tr = &traceSize{
			size: sz,
		}
		s.sizes[token] = tr
	}
	tr.timestamp = time.Now()
	tr.size += sz

	return tr.size <= max
}

func (s *traceSizes) ClearIdle(idleSince time.Time) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for token, tr := range s.sizes {
		if tr.timestamp.Before(idleSince) {
			delete(s.sizes, token)
		}
	}
}
