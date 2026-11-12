package store

import (
	"container/list"
	"maps"
	"sync"
	"time"

	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

// SpanStore matches direct parent/child spans independently of their kinds.
// Parents remain cached after matching, since a parent may have several children.
// Both cached spans and unresolved child links are bounded by maxItems and ttl.
// Callbacks run under the mutex; Edge pointers must not escape their callbacks.
type SpanStore struct {
	mu         sync.Mutex
	spans      map[edgeKey]*cachedSpan
	waiting    map[edgeKey]map[edgeKey]*cachedSpan
	order      list.List
	ttl        time.Duration
	maxItems   int
	onComplete Callback
	onExpire   Callback
	onDropped  func()
}

type cachedSpan struct {
	key, parent      edgeKey
	waitingFor       edgeKey
	data             Edge
	serverDimensions map[string]string
	filtered         bool
	parentMatched    bool
	hasChildren      bool
	expires          time.Time
}

func NewSpanStore(ttl time.Duration, maxItems int, onComplete, onExpire Callback, onDropped func()) *SpanStore {
	return &SpanStore{
		spans:   make(map[edgeKey]*cachedSpan),
		waiting: make(map[edgeKey]map[edgeKey]*cachedSpan),
		ttl:     ttl, maxItems: maxItems, onComplete: onComplete, onExpire: onExpire, onDropped: onDropped,
	}
}

// AddSpan copies IDs and records only the metadata supplied by update, never the
// input span itself. A filtered span is retained as a bounded tombstone so that
// neither of its adjacent relationships can become an inferred virtual edge.
// update supplies the span's service, duration and attributes in the client-side
// Edge fields, and its server-side dimensions in the separate map. ServerService
// is reserved for inferred database destinations. These views let the same span
// participate as the parent of one relationship and the child of another.
// Repeated IDs are ignored within the cache lifetime to avoid counting an edge twice.
func (s *SpanStore) AddSpan(traceID, spanID, parentSpanID []byte, filtered bool, update func(*Edge, map[string]string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.expire(now)
	key := edgeKeyFromBytes(traceID, spanID)
	if _, exists := s.spans[key]; exists {
		return nil
	}
	if len(s.spans) >= s.maxItems {
		return ErrTooManyItems
	}
	node := &cachedSpan{
		key: key, parent: edgeKeyFromBytes(traceID, parentSpanID),
		filtered: filtered, expires: now.Add(s.ttl),
	}
	if !filtered {
		node.data.Dimensions = make(map[string]string)
		node.serverDimensions = make(map[string]string)
		update(&node.data, node.serverDimensions)
	}
	s.order.PushBack(node)
	s.spans[key] = node

	// Database spans can describe a complete external edge without a child span.
	if !filtered && node.data.ServerService != "" {
		s.onComplete(&node.data)
		node.hasChildren = true
	}
	if !node.parent.root {
		if parent, exists := s.spans[node.parent]; exists {
			s.match(parent, node)
		} else {
			s.waitFor(node.parent, node)
		}
	}
	if children := s.waiting[key]; children != nil {
		delete(s.waiting, key)
		for _, child := range children {
			s.match(node, child)
		}
	}
	return nil
}

// waitFor indexes a child by the missing direct parent.
func (s *SpanStore) waitFor(key edgeKey, child *cachedSpan) {
	children := s.waiting[key]
	if children == nil {
		children = make(map[edgeKey]*cachedSpan)
		s.waiting[key] = children
	}
	children[child.key] = child
	child.waitingFor = key
}

func (s *SpanStore) match(parent, child *cachedSpan) {
	child.parentMatched = true
	parent.hasChildren = true
	if parent.filtered || child.filtered {
		if s.onDropped != nil {
			s.onDropped()
		}
		return
	}
	if parent.data.ClientService == child.data.ClientService {
		return
	}
	edge := Edge{
		ClientService: parent.data.ClientService, ServerService: child.data.ClientService,
		ClientLatencySec: parent.data.ClientLatencySec, ServerLatencySec: child.data.ClientLatencySec,
		ClientEndTimeUnixNano:   parent.data.ClientEndTimeUnixNano,
		ServerStartTimeUnixNano: child.data.ServerStartTimeUnixNano,
		Failed:                  parent.data.Failed || child.data.Failed,
		SpanMultiplier:          parent.data.SpanMultiplier,
		SpanKind:                child.data.SpanKind,
		Dimensions:              make(map[string]string, len(parent.data.Dimensions)+len(child.serverDimensions)),
	}
	// Parent and child belong to the same trace. This alias is valid only during
	// the synchronous callback, before the cached parent can be expired.
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

func (s *SpanStore) Expire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire(time.Now())
}

func (s *SpanStore) expire(now time.Time) {
	for front := s.order.Front(); front != nil; front = s.order.Front() {
		node := front.Value.(*cachedSpan)
		if now.Before(node.expires) {
			return
		}
		if !node.filtered {
			// An unmatched incoming relationship is represented like a server side in
			// the strict store. Root server/consumer spans retain virtual-client inference.
			if !node.parentMatched && (!node.parent.root || node.data.SpanKind == tracev1.Span_SPAN_KIND_SERVER || node.data.SpanKind == tracev1.Span_SPAN_KIND_CONSUMER) {
				edge := node.data
				edge.key = node.parent
				edge.ServerService = edge.ClientService
				edge.ServerLatencySec = edge.ClientLatencySec
				edge.ClientService = ""
				edge.ClientLatencySec = 0
				edge.Dimensions = maps.Clone(node.serverDimensions)
				s.onExpire(&edge)
			}
			// Only client/producer spans describe outgoing requests to uninstrumented peers.
			if !node.hasChildren && (node.data.SpanKind == tracev1.Span_SPAN_KIND_CLIENT || node.data.SpanKind == tracev1.Span_SPAN_KIND_PRODUCER) {
				edge := node.data
				edge.Dimensions = maps.Clone(node.data.Dimensions)
				s.onExpire(&edge)
			}
		}
		if children := s.waiting[node.waitingFor]; children != nil {
			delete(children, node.key)
			if len(children) == 0 {
				delete(s.waiting, node.waitingFor)
			}
		}
		delete(s.spans, node.key)
		s.order.Remove(front)
	}
}
