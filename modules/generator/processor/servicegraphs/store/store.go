package store

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var ErrTooManyItems = errors.New("too many items")

var _ Store = (*store)(nil)

type store struct {
	l   *list.List
	mtx sync.Mutex
	m   map[string]*list.Element

	onComplete Callback
	onExpire   Callback

	ttl      time.Duration
	maxItems int
}

var edgePool = sync.Pool{
	New: func() interface{} {
		return &Edge{
			Dimensions: make(map[string]string),
		}
	},
}

// GrabEdge returns a new Edge from the pool, clearing its state and setting the key and expiration.
func (s *store) GrabEdge(key string) *Edge {
	edge := edgePool.Get().(*Edge)
	zeroStateEdge(edge)
	edge.key = key
	edge.expiration = time.Now().Add(s.ttl).Unix()
	return edge
}

// ReturnEdge returns an Edge to the pool.
func (s *store) ReturnEdge(e *Edge) {
	edgePool.Put(e)
}

// NewStore creates a Store to build service graphs. The store caches edges, each representing a
// request between two services. Once an edge is complete its metrics can be collected. Edges that
// have not found their pair are deleted after ttl time.
func NewStore(ttl time.Duration, maxItems int, onComplete, onExpire Callback) Store {
	s := &store{
		l: list.New(),
		m: make(map[string]*list.Element),

		onComplete: onComplete,
		onExpire:   onExpire,

		ttl:      ttl,
		maxItems: maxItems,
	}

	return s
}

func (s *store) len() int {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.l.Len()
}

// tryEvictHead checks if the oldest item (head of list) can be evicted and will delete it if so.
// Returns true if the head was evicted.
//
// Must be called holding lock.
func (s *store) tryEvictHead() bool {
	head := s.l.Front()
	if head == nil {
		// list is empty
		return false
	}

	headEdge := head.Value.(*Edge)
	if !headEdge.isExpired() {
		return false
	}

	s.onExpire(headEdge)
	delete(s.m, headEdge.key)
	s.l.Remove(head)

	return true
}

// UpsertEdge fetches an Edge from the store and updates it using the given callback. If the Edge
// doesn't exist yet, it creates a new one with the default TTL.
// If the Edge is complete after applying the callback, it's completed and removed.
func (s *store) UpsertEdge(key string, update Callback) (isNew bool, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if storedEdge, ok := s.m[key]; ok {
		edge := storedEdge.Value.(*Edge)
		update(edge)

		if edge.isComplete() {
			s.onComplete(edge)
			delete(s.m, key)
			s.l.Remove(storedEdge)
		}

		return false, nil
	}

	edge := s.GrabEdge(key)
	update(edge)

	if edge.isComplete() {
		s.onComplete(edge)
		return true, nil
	}

	// Check we can add new edges
	if s.l.Len() >= s.maxItems {
		// todo: try to evict expired items
		return false, ErrTooManyItems
	}

	ele := s.l.PushBack(edge)
	s.m[key] = ele

	return true, nil
}

// Expire evicts all expired items in the store.
func (s *store) Expire() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for s.tryEvictHead() {
	}
}
