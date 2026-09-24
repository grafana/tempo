package store

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrTooManyItems    = errors.New("too many items")
	ErrDroppedSpanSide = errors.New("dropped span side")
)

// Store builds service graph edges from paired client/server spans.
type Store struct {
	mtx sync.Mutex
	m   map[edgeKey]*Edge
	d   map[droppedSpanSideKey]int64

	head *Edge
	tail *Edge

	onComplete Callback
	onExpire   Callback

	ttl      time.Duration
	maxItems int

	droppedSpanSideOverflowCounter prometheus.Counter
}

type droppedSpanSideKey struct {
	key  edgeKey
	side Side
}

const (
	maxTraceIDLen = 16
	maxSpanIDLen  = 8
)

// edgeKey keeps valid OTLP IDs inline. Oversized malformed IDs use an encoded
// fallback so their full contents still participate in matching.
type edgeKey struct {
	traceID  [maxTraceIDLen]byte
	spanID   [maxSpanIDLen]byte
	fallback string

	traceIDLen uint8
	spanIDLen  uint8
	root       bool
}

func edgeKeyFromBytes(traceID, spanID []byte) edgeKey {
	key := edgeKey{root: len(spanID) == 0}
	if len(traceID) > maxTraceIDLen || len(spanID) > maxSpanIDLen {
		key.fallback = encodeKey(traceID, spanID)
		return key
	}

	key.traceIDLen = uint8(len(traceID))
	key.spanIDLen = uint8(len(spanID))
	copy(key.traceID[:], traceID)
	copy(key.spanID[:], spanID)
	return key
}

func (k edgeKey) String() string {
	if k.fallback != "" {
		return k.fallback
	}
	return encodeKey(k.traceID[:k.traceIDLen], k.spanID[:k.spanIDLen])
}

// NewStore creates a Store to build service graphs. The store caches edges, each representing a
// request between two services. Once an edge is complete its metrics can be collected. Edges that
// have not found their pair are deleted after ttl time.
func NewStore(ttl time.Duration, maxItems int, onComplete, onExpire Callback, droppedSpanSideOverflowCounter prometheus.Counter) *Store {
	s := &Store{
		m: make(map[edgeKey]*Edge),
		d: make(map[droppedSpanSideKey]int64),

		onComplete: onComplete,
		onExpire:   onExpire,

		ttl:      ttl,
		maxItems: maxItems,

		droppedSpanSideOverflowCounter: droppedSpanSideOverflowCounter,
	}

	return s
}

func (s *Store) len() int {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return len(s.m)
}

// tryEvictHead checks if the oldest item (head of list) can be evicted and will delete it if so.
// Returns true if the head was evicted.
//
// Must be called holding lock.
func (s *Store) tryEvictHead() bool {
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
func (s *Store) deleteEdge(edge *Edge) {
	delete(s.m, edge.key)
	s.removeEdge(edge)
	s.returnEdge(edge)
}

// UpsertEdgeFromBytes fetches an Edge from the store and updates it using the
// given callback. If the Edge doesn't exist yet, it creates a new one with the
// default TTL. If the Edge is complete after applying the callback, it's
// completed and removed.
func (s *Store) UpsertEdgeFromBytes(traceID, spanID []byte, side Side, update Callback) (isNew bool, err error) {
	return s.upsertEdge(edgeKeyFromBytes(traceID, spanID), side, update)
}

func (s *Store) upsertEdge(key edgeKey, side Side, update Callback) (isNew bool, err error) {
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
	if len(s.m) >= s.maxItems {
		// todo: try to evict expired items
		s.returnEdge(edge)
		return false, ErrTooManyItems
	}

	s.pushBack(edge)
	s.m[key] = edge

	return true, nil
}

func (s *Store) pushBack(edge *Edge) {
	edge.prev = s.tail
	edge.next = nil
	if s.tail != nil {
		s.tail.next = edge
	} else {
		s.head = edge
	}
	s.tail = edge
}

func (s *Store) removeEdge(edge *Edge) {
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
}

func encodedKeyLen(k1, k2 []byte) int {
	return hex.EncodedLen(len(k1)) + 1 + hex.EncodedLen(len(k2))
}

func encodeKey(k1, k2 []byte) string {
	buf := make([]byte, encodedKeyLen(k1, k2))
	k1Len := hex.EncodedLen(len(k1))
	hex.Encode(buf[:k1Len], k1)
	buf[k1Len] = '-'
	hex.Encode(buf[k1Len+1:], k2)
	return string(buf)
}

// Expire evicts all expired items in the store.
func (s *Store) Expire() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for s.tryEvictHead() {
	}

	s.expireDroppedSpanSides()
}

// AddDroppedSpanSideFromBytes records a filtered edge side using trace/span ID bytes.
func (s *Store) AddDroppedSpanSideFromBytes(traceID, spanID []byte, side Side) bool {
	return s.addDroppedSpanSide(edgeKeyFromBytes(traceID, spanID), side)
}

func (s *Store) addDroppedSpanSide(key edgeKey, side Side) bool {
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

// HasDroppedSpanSideFromBytes reports whether a trace/span ID side is recorded as dropped.
func (s *Store) HasDroppedSpanSideFromBytes(traceID, spanID []byte, side Side) bool {
	return s.hasDroppedSpanSide(edgeKeyFromBytes(traceID, spanID), side)
}

func (s *Store) hasDroppedSpanSide(key edgeKey, side Side) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.d[droppedSpanSideKey{
		key:  key,
		side: side,
	}]
	return ok
}

func (s *Store) expireDroppedSpanSides() {
	now := time.Now().UnixNano()
	for k, expiration := range s.d {
		if now >= expiration {
			delete(s.d, k)
		}
	}
}

func (s *Store) hasDroppedCounterpart(key edgeKey, side Side) bool {
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
func (s *Store) grabEdge(key edgeKey) *Edge {
	edge := edgePool.Get().(*Edge)
	resetEdge(edge)
	edge.key = key
	edge.expiration = time.Now().Add(s.ttl).Unix()
	return edge
}

// returnEdge returns an Edge to the pool.
func (s *Store) returnEdge(e *Edge) {
	// Do not retain request-owned keys or malformed request-sized trace IDs in
	// the global pool while an edge is idle.
	e.key = edgeKey{}
	if cap(e.traceID) > maxTraceIDLen {
		e.traceID = nil
	}
	edgePool.Put(e)
}

func getEdgeSide(e *Edge) Side {
	if e.ServerService != "" {
		return Server
	}
	return Client
}
