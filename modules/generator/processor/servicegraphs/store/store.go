package store

import (
	"container/list"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrTooManyItems    = errors.New("too many items")
	ErrDroppedSpanSide = errors.New("dropped span side")
)

var _ Store = (*store)(nil)

type store struct {
	l   *list.List
	mtx sync.Mutex
	m   map[string]*list.Element
	d   map[droppedSpanSideKey]int64

	onComplete Callback
	onExpire   Callback

	ttl      time.Duration
	maxItems int

	droppedSpanSideOverflowCounter prometheus.Counter
}

type droppedSpanSideKey struct {
	key  string
	side Side
}

// NewStore creates a Store to build service graphs. The store caches edges, each representing a
// request between two services. Once an edge is complete its metrics can be collected. Edges that
// have not found their pair are deleted after ttl time.
func NewStore(ttl time.Duration, maxItems int, onComplete, onExpire Callback, droppedSpanSideOverflowCounter prometheus.Counter) Store {
	s := &store{
		l: list.New(),
		m: make(map[string]*list.Element),
		d: make(map[droppedSpanSideKey]int64),

		onComplete: onComplete,
		onExpire:   onExpire,

		ttl:      ttl,
		maxItems: maxItems,

		droppedSpanSideOverflowCounter: droppedSpanSideOverflowCounter,
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
	s.deleteEdge(head)

	return true
}

// deleteEdge removes an edge from the map/list and returns it to the pool.
// Must be called holding lock.
func (s *store) deleteEdge(ele *list.Element) {
	edge := ele.Value.(*Edge)
	delete(s.m, edge.key)
	s.l.Remove(ele)
	s.returnEdge(edge)
}

// UpsertEdge fetches an Edge from the store and updates it using the given callback. If the Edge
// doesn't exist yet, it creates a new one with the default TTL.
// If the Edge is complete after applying the callback, it's completed and removed.
func (s *store) UpsertEdge(key string, side Side, update Callback) (isNew bool, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if storedEdge, ok := s.m[key]; ok {
		edge := storedEdge.Value.(*Edge)
		update(edge)

		if edge.isComplete() {
			s.onComplete(edge)
			s.deleteEdge(storedEdge)
		}

		return false, nil
	}

	if s.hasDroppedCounterpart(key, side) {
		return true, ErrDroppedSpanSide
	}

	edge := s.grabEdge(key)
	update(edge)

	if edge.isComplete() {
		s.onComplete(edge)
		s.returnEdge(edge)
		return true, nil
	}

	// Check we can add new edges
	if s.l.Len() >= s.maxItems {
		// todo: try to evict expired items
		s.returnEdge(edge)
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

	s.expireDroppedSpanSides()
}

// This cache is best-effort metadata for dropped-span correlation.
func (s *store) AddDroppedSpanSide(key string, side Side) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	droppedCounterpartEdge := false
	if storedEdge, ok := s.m[key]; ok {
		edge := storedEdge.Value.(*Edge)
		// If a counterpart edge is already buffered, drop it immediately instead of
		// waiting for TTL expiration.
		if !edge.isComplete() && getEdgeSide(edge) != side {
			s.deleteEdge(storedEdge)
			droppedCounterpartEdge = true
		}
	}

	k := droppedSpanSideKey{
		key:  key,
		side: side,
	}

	// Refresh TTL for existing entries
	if _, ok := s.d[k]; ok {
		s.d[k] = time.Now().Add(s.ttl).UnixNano()
		return droppedCounterpartEdge
	}

	if len(s.d) >= s.maxItems {
		s.droppedSpanSideOverflowCounter.Inc()
		return droppedCounterpartEdge
	}

	s.d[k] = time.Now().Add(s.ttl).UnixNano()
	return droppedCounterpartEdge
}

func (s *store) HasDroppedSpanSide(key string, side Side) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.d[droppedSpanSideKey{
		key:  key,
		side: side,
	}]
	return ok
}

func (s *store) expireDroppedSpanSides() {
	now := time.Now().UnixNano()
	for k, expiration := range s.d {
		if now >= expiration {
			delete(s.d, k)
		}
	}
}

func (s *store) hasDroppedCounterpart(key string, side Side) bool {
	counterpart := Client
	if side == Client {
		counterpart = Server
	}

	_, ok := s.d[droppedSpanSideKey{key: key, side: counterpart}]
	return ok
}

var edgePool = sync.Pool{
	New: func() interface{} {
		return &Edge{
			Dimensions: make(map[string]string),
		}
	},
}

// grabEdge returns a new Edge from the pool, clearing its state and setting the key and expiration.
func (s *store) grabEdge(key string) *Edge {
	edge := edgePool.Get().(*Edge)
	resetEdge(edge)
	edge.key = key
	edge.expiration = time.Now().Add(s.ttl).Unix()
	return edge
}

// returnEdge returns an Edge to the pool.
func (s *store) returnEdge(e *Edge) {
	edgePool.Put(e)
}

func getEdgeSide(e *Edge) Side {
	if e.ServerService != "" {
		return Server
	}
	return Client
}
