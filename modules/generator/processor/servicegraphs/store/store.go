package store

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrTooManyItems    = errors.New("too many items")
	ErrDroppedSpanSide = errors.New("dropped span side")
)

var _ Store = (*store)(nil)

type store struct {
	mtx sync.Mutex
	m   map[string]*Edge
	d   map[droppedSpanSideKey]int64

	head  *Edge
	tail  *Edge
	items int

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
		m: make(map[string]*Edge),
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

	return s.items
}

// tryEvictHead checks if the oldest item (head of list) can be evicted and will delete it if so.
// Returns true if the head was evicted.
//
// Must be called holding lock.
func (s *store) tryEvictHead() bool {
	if s.head == nil {
		// list is empty
		return false
	}

	if !s.head.isExpired() {
		return false
	}

	s.onExpire(s.head)
	s.deleteEdge(s.head)

	return true
}

// deleteEdge removes an edge from the map/list and returns it to the pool.
// Must be called holding lock.
func (s *store) deleteEdge(edge *Edge) {
	delete(s.m, edge.key)
	s.removeEdge(edge)
	s.returnEdge(edge)
}

// UpsertEdge fetches an Edge from the store and updates it using the given callback. If the Edge
// doesn't exist yet, it creates a new one with the default TTL.
// If the Edge is complete after applying the callback, it's completed and removed.
func (s *store) UpsertEdge(key string, side Side, update Callback) (isNew bool, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if edge, ok := s.m[key]; ok {
		update(edge)

		if edge.isComplete() {
			s.onComplete(edge)
			s.deleteEdge(edge)
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
	if s.items >= s.maxItems {
		// todo: try to evict expired items
		s.returnEdge(edge)
		return false, ErrTooManyItems
	}

	s.pushBack(edge)
	s.m[key] = edge

	return true, nil
}

func (s *store) UpsertEdgeFromBytes(traceID, spanID []byte, side Side, update Callback) (isNew bool, err error) {
	return UpsertEdgeFromBytesWith(s, traceID, spanID, side, update, func(edge *Edge, update Callback) {
		update(edge)
	})
}

// UpsertEdgeFromBytesWith is like Store.UpsertEdgeFromBytes, but passes typed
// state to a top-level update function so callers can avoid per-call closures.
func UpsertEdgeFromBytesWith[T any](st Store, traceID, spanID []byte, side Side, state T, update func(*Edge, T)) (isNew bool, err error) {
	s, ok := st.(*store)
	if !ok {
		return st.UpsertEdgeFromBytes(traceID, spanID, side, func(edge *Edge) {
			update(edge, state)
		})
	}
	return upsertEdgeFromBytesWith(s, traceID, spanID, side, state, update)
}

// maxStackKeyLen is the largest encoded edge key built in a stack buffer
// before falling back to the heap-allocating path. A spec-compliant key is
// hex(16-byte trace ID) + "-" + hex(8-byte span ID) = 49 bytes, so 64 covers
// normal IDs with headroom while keeping the buffer cheap to stack-allocate.
const maxStackKeyLen = 64

func upsertEdgeFromBytesWith[T any](s *store, traceID, spanID []byte, side Side, state T, update func(*Edge, T)) (isNew bool, err error) {
	encodedLen := encodedKeyLen(traceID, spanID)
	if encodedLen > maxStackKeyLen {
		return s.UpsertEdge(encodeKey(traceID, spanID), side, func(edge *Edge) {
			update(edge, state)
		})
	}

	var buf [maxStackKeyLen]byte
	key := encodeKeyToString(buf[:encodedLen], traceID, spanID)

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if edge, ok := s.m[key]; ok {
		update(edge, state)

		if edge.isComplete() {
			s.onComplete(edge)
			s.deleteEdge(edge)
		}

		return false, nil
	}

	if s.hasDroppedCounterpart(key, side) {
		return true, ErrDroppedSpanSide
	}

	edge := s.grabEdgeFromBytes(traceID, spanID)
	update(edge, state)

	if edge.isComplete() {
		s.onComplete(edge)
		s.returnEdge(edge)
		return true, nil
	}

	// Check we can add new edges
	if s.items >= s.maxItems {
		// todo: try to evict expired items
		s.returnEdge(edge)
		return false, ErrTooManyItems
	}

	s.pushBack(edge)
	s.m[edge.key] = edge

	return true, nil
}

func (s *store) pushBack(edge *Edge) {
	edge.prev = s.tail
	edge.next = nil
	if s.tail != nil {
		s.tail.next = edge
	} else {
		s.head = edge
	}
	s.tail = edge
	s.items++
}

func (s *store) removeEdge(edge *Edge) {
	if edge.prev != nil {
		edge.prev.next = edge.next
	} else {
		s.head = edge.next
	}
	if edge.next != nil {
		edge.next.prev = edge.prev
	} else {
		s.tail = edge.prev
	}
	edge.prev = nil
	edge.next = nil
	s.items--
}

func encodedKeyLen(k1, k2 []byte) int {
	return hex.EncodedLen(len(k1)) + 1 + hex.EncodedLen(len(k2))
}

func encodeKey(k1, k2 []byte) string {
	buf := make([]byte, encodedKeyLen(k1, k2))
	return encodeKeyToString(buf, k1, k2)
}

func encodeKeyToString(buf []byte, k1, k2 []byte) string {
	k1Len := hex.EncodedLen(len(k1))
	hex.Encode(buf[:k1Len], k1)
	buf[k1Len] = '-'
	hex.Encode(buf[k1Len+1:], k2)
	// The caller owns buf and keeps it immutable for as long as the returned
	// key is in use. This avoids copying servicegraph keys on the hot path.
	return unsafe.String(unsafe.SliceData(buf), len(buf)) // nosemgrep
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
	if edge, ok := s.m[key]; ok {
		// If a counterpart edge is already buffered, drop it immediately instead of
		// waiting for TTL expiration.
		if !edge.isComplete() && getEdgeSide(edge) != side {
			s.deleteEdge(edge)
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

func (s *store) grabEdgeFromBytes(traceID, spanID []byte) *Edge {
	edge := edgePool.Get().(*Edge)
	resetEdge(edge)
	setEdgeKeyFromBytes(edge, traceID, spanID)
	edge.expiration = time.Now().Add(s.ttl).Unix()
	return edge
}

// setEdgeKeyFromBytes encodes the key into the edge's reusable keyBuf and
// aliases edge.key over it. It runs on freshly grabbed (possibly recycled)
// edges, so it must fully overwrite any previous key contents.
func setEdgeKeyFromBytes(edge *Edge, traceID, spanID []byte) {
	encodedLen := encodedKeyLen(traceID, spanID)
	if cap(edge.keyBuf) < encodedLen {
		edge.keyBuf = make([]byte, encodedLen)
	}
	edge.keyBuf = edge.keyBuf[:encodedLen]
	edge.key = encodeKeyToString(edge.keyBuf, traceID, spanID)
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
