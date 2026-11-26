package store

import (
	"container/list"
	"maps"
	"sync"
	"time"

	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

// SpanStore settles bounded span windows before emitting relationships. This
// allows shared client/server IDs to arrive in either order without counting an
// edge that a later counterpart would change. Callbacks run under the mutex;
// Edge pointers must not escape their callbacks. Expire must be called periodically;
// AddSpan never scans the complete cache on the per-span ingestion path.
type SpanStore struct {
	mu         sync.Mutex
	spans      map[spanIdentity]*cachedSpan
	traces     map[edgeKey]*spanWindow
	order      list.List
	ttl        time.Duration
	now        func() time.Time
	maxItems   int
	onComplete Callback
	onExpire   Callback
	onDropped  func()
}

type spanIdentity struct {
	id   edgeKey
	kind tracev1.Span_SpanKind
}

type spanWindow struct {
	key   edgeKey
	nodes []*cachedSpan
}

type cachedSpan struct {
	key               spanIdentity
	parent            edgeKey
	parentNode        *cachedSpan
	data              Edge
	serverDimensions  map[string]string
	filtered          bool
	hasRemoteChild    bool
	outgoingCommitted bool
	incomingDone      bool
	expires           time.Time
	window            *spanWindow
	element           *list.Element
}

func NewSpanStore(ttl time.Duration, maxItems int, onComplete, onExpire Callback, onDropped func()) *SpanStore {
	return &SpanStore{spans: make(map[spanIdentity]*cachedSpan), traces: make(map[edgeKey]*spanWindow), ttl: ttl, now: time.Now, maxItems: maxItems, onComplete: onComplete, onExpire: onExpire, onDropped: onDropped}
}

// AddSpan copies IDs and the metadata supplied by update. Filtered spans remain
// bounded tombstones, preventing either adjacent relationship from being inferred.
// Each span expires after ttl; duplicates of the same ID and kind
// do not update metadata or count twice within that window.
func (s *SpanStore) AddSpan(traceID, spanID, parentSpanID []byte, kind tracev1.Span_SpanKind, filtered bool, update func(*Edge, map[string]string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	key := spanIdentity{edgeKeyFromBytes(traceID, spanID), kind}
	if _, exists := s.spans[key]; exists {
		return nil
	}
	if len(s.spans) >= s.maxItems {
		return ErrTooManyItems
	}
	node := &cachedSpan{key: key, parent: edgeKeyFromBytes(traceID, parentSpanID), filtered: filtered, expires: now.Add(s.ttl)}
	node.data.SpanKind = kind
	if !filtered {
		node.data.Dimensions = make(map[string]string)
		node.serverDimensions = make(map[string]string)
		update(&node.data, node.serverDimensions)
	}
	windowKey := edgeKeyFromBytes(traceID, nil)
	window := s.traces[windowKey]
	if window == nil {
		window = &spanWindow{key: windowKey}
		s.traces[windowKey] = window
	}
	node.window = window
	node.element = s.order.PushBack(node)
	window.nodes = append(window.nodes, node)
	s.spans[key] = node
	return nil
}

// Expire emits settled trace windows and releases their metadata. Taking now
// explicitly also permits deterministic expiry tests without sleeps.
func (s *SpanStore) Expire(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire(now)
}

func (s *SpanStore) expire(now time.Time) {
	for front := s.order.Front(); front != nil; front = s.order.Front() {
		first := front.Value.(*cachedSpan)
		if now.Before(first.expires) {
			return
		}
		window := first.window
		s.complete(window, now)
		old := window.nodes
		remaining := old[:0]
		for _, node := range old {
			node.parentNode = nil
			if !now.Before(node.expires) {
				delete(s.spans, node.key)
				s.order.Remove(node.element)
			} else {
				remaining = append(remaining, node)
			}
		}
		clear(old[len(remaining):])
		window.nodes = remaining
		if len(remaining) == 0 {
			delete(s.traces, window.key)
		}
	}
}

// complete resolves a trace snapshot in linear time. A shared server endpoint
// belongs to its same-ID client; other children prefer the server endpoint of a
// shared ID. A filtered endpoint wins conservatively so filtering cannot expose
// an inferred replacement relationship.
func (s *SpanStore) complete(window *spanWindow, now time.Time) {
	parents := make(map[edgeKey]*cachedSpan, len(window.nodes))
	clients := make(map[edgeKey]*cachedSpan)
	for _, node := range window.nodes {
		if node.key.kind == tracev1.Span_SPAN_KIND_CLIENT {
			clients[node.key.id] = node
		}
		existing := parents[node.key.id]
		if existing == nil || (!existing.filtered && (node.filtered || node.key.kind == tracev1.Span_SPAN_KIND_SERVER)) {
			parents[node.key.id] = node
		}
	}
	for _, node := range window.nodes {
		node.hasRemoteChild = node.outgoingCommitted
		node.parentNode = nil
	}
	for _, node := range window.nodes {
		if node.incomingDone {
			continue
		}
		parent := parents[node.parent]
		if node.key.kind == tracev1.Span_SPAN_KIND_SERVER && clients[node.key.id] != nil {
			parent = clients[node.key.id]
		}
		if parent == node {
			parent = nil
		}
		node.parentNode = parent
		if parent == nil {
			continue
		}
		settled := !now.Before(parent.expires) || !now.Before(node.expires)
		if settled {
			node.incomingDone = true
		}
		if parent.filtered || node.filtered {
			parent.hasRemoteChild = true
			if settled {
				parent.outgoingCommitted = true
			}
			if settled && s.onDropped != nil {
				s.onDropped()
			}
			continue
		}
		if parent.data.ClientService != node.data.ClientService {
			parent.hasRemoteChild = true
			if settled {
				parent.outgoingCommitted = true
				s.match(parent, node)
			}
		}
	}
	// Mark database destinations and propagate observed remote destinations through
	// local ancestors. Each node is marked once, avoiding quadratic chain scans.
	for _, node := range window.nodes {
		if !node.filtered && node.data.ServerService != "" {
			if !now.Before(node.expires) {
				s.onComplete(&node.data)
				node.outgoingCommitted = true
			}
			node.hasRemoteChild = true
		}
	}
	for _, node := range window.nodes {
		if !node.hasRemoteChild {
			continue
		}
		for child, parent := node, node.parentNode; parent != nil && !parent.hasRemoteChild && parent.data.ClientService == child.data.ClientService; child, parent = parent, parent.parentNode {
			parent.hasRemoteChild = true
			if child.outgoingCommitted {
				parent.outgoingCommitted = true
			}
		}
	}
	for _, node := range window.nodes {
		if node.filtered || now.Before(node.expires) {
			continue
		}
		if !node.incomingDone && node.parentNode == nil && (!node.parent.root || node.data.SpanKind == tracev1.Span_SPAN_KIND_SERVER || node.data.SpanKind == tracev1.Span_SPAN_KIND_CONSUMER) {
			edge := node.data
			edge.key = node.parent
			edge.ServerService = edge.ClientService
			edge.ServerLatencySec = edge.ClientLatencySec
			edge.ClientService = ""
			edge.ClientLatencySec = 0
			edge.Dimensions = maps.Clone(node.serverDimensions)
			s.onExpire(&edge)
		}
		if !node.hasRemoteChild && (node.data.SpanKind == tracev1.Span_SPAN_KIND_CLIENT || node.data.SpanKind == tracev1.Span_SPAN_KIND_PRODUCER) {
			edge := node.data
			edge.Dimensions = maps.Clone(node.data.Dimensions)
			s.onExpire(&edge)
		}
	}
}

func (s *SpanStore) match(parent, child *cachedSpan) {
	edge := Edge{
		ClientService: parent.data.ClientService, ServerService: child.data.ClientService,
		ClientLatencySec: parent.data.ClientLatencySec, ServerLatencySec: child.data.ClientLatencySec,
		ClientEndTimeUnixNano: parent.data.ClientEndTimeUnixNano, ServerStartTimeUnixNano: child.data.ServerStartTimeUnixNano,
		Failed: parent.data.Failed || child.data.Failed, SpanMultiplier: parent.data.SpanMultiplier, SpanKind: child.data.SpanKind,
		Dimensions: make(map[string]string, len(parent.data.Dimensions)+len(child.serverDimensions)),
	}
	edge.traceID = parent.data.TraceID()
	for k, v := range parent.data.Dimensions {
		edge.Dimensions[k] = v
	}
	for k, v := range child.serverDimensions {
		edge.Dimensions[k] = v
	}
	if parent.data.SpanKind == tracev1.Span_SPAN_KIND_PRODUCER && child.data.SpanKind == tracev1.Span_SPAN_KIND_CONSUMER {
		edge.ConnectionType = MessagingSystem
	}
	s.onComplete(&edge)
}
