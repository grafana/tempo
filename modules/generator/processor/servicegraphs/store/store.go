package store

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var (
	ErrTooManyItems = errors.New("too many items")
)

var _ Store = (*store)(nil)

type store struct {
	l   *list.List
	mtx *sync.RWMutex
	m   map[string]*list.Element

	evictCallback Callback
	ttl           time.Duration
	maxItems      int
}

func NewStore(ttl time.Duration, maxItems int, evictCallback Callback) Store {
	s := &store{
		l:   list.New(),
		mtx: &sync.RWMutex{},
		m:   make(map[string]*list.Element),

		evictCallback: evictCallback,
		ttl:           ttl,
		maxItems:      maxItems,
	}

	return s
}

func (s *store) len() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.l.Len()
}

// shouldEvictHead checks if the oldest item (head of list) has expired and should be evicted.
// Returns true if the item has expired or is completed, false otherwise.
//
// Must be called under lock.
func (s *store) shouldEvictHead() bool {
	h := s.l.Front()
	if h == nil {
		return false
	}
	edge := h.Value.(*Edge)
	return edge.IsCompleted() || edge.IsExpired()
}

// evictHead removes the head from the store (and map).
// It also collects metrics for the evicted Edge.
//
// Must be called under lock.
func (s *store) evictHead() {
	front := s.l.Front().Value.(*Edge)
	s.evictEdge(front.key)
}

// EvictEdge evicts and Edge under lock
func (s *store) EvictEdge(key string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.evictEdge(key)
}

// evictEdge removes the Edge from the store (and map).
// It also collects metrics for the evicted Edge.
//
// Must be called under lock.
func (s *store) evictEdge(key string) {
	ele := s.m[key]
	if ele == nil { // it may already have been processed
		return
	}

	edge := ele.Value.(*Edge)
	s.evictCallback(edge)

	delete(s.m, key)
	s.l.Remove(ele)
}

// UpsertEdge fetches an Edge from the store.
// If the Edge doesn't exist, it creates a new one with the default TTL.
func (s *store) UpsertEdge(k string, cb Callback) (*Edge, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if storedEdge, ok := s.m[k]; ok {
		edge := storedEdge.Value.(*Edge)
		cb(edge)
		return edge, nil
	}

	if s.l.Len() >= s.maxItems {
		// todo: try to evict expired items
		return nil, ErrTooManyItems
	}

	newEdge := NewEdge(k, s.ttl)
	ele := s.l.PushBack(newEdge)
	s.m[k] = ele
	cb(newEdge)

	return newEdge, nil
}

// Expire evicts all expired items in the store.
func (s *store) Expire() {
	s.mtx.RLock()
	if !s.shouldEvictHead() {
		s.mtx.RUnlock()
		return
	}
	s.mtx.RUnlock()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	for s.shouldEvictHead() {
		s.evictHead()
	}
}
