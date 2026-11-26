package store

import (
	"testing"
	"time"

	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

// Zipkin shared client/server IDs describe different service endpoints. The
// relaxed matcher must retain both, independently of which side arrives first.
func TestReviewSharedSpanIDPreservesBothServices(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "client first"
		if reverse {
			name = "server first"
		}
		t.Run(name, func(t *testing.T) {
			var pairs []string
			s := NewSpanStore(time.Hour, 10, func(e *Edge) { pairs = append(pairs, e.ClientService+"->"+e.ServerService) }, func(*Edge) {}, nil)
			add := func(service string, id, parent byte, kind tracev1.Span_SpanKind) {
				var parentID []byte
				if parent != 0 {
					parentID = reviewID(8, parent)
				}
				require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, id), parentID, kind, false, func(e *Edge, _ map[string]string) {
					e.ClientService = service
					e.SpanKind = kind
					e.SetTraceID(reviewID(16, 1))
				}))
			}
			add("A", 1, 0, tracev1.Span_SPAN_KIND_INTERNAL)
			if reverse {
				add("B", 2, 1, tracev1.Span_SPAN_KIND_SERVER)
				add("gateway", 2, 1, tracev1.Span_SPAN_KIND_CLIENT)
			} else {
				add("gateway", 2, 1, tracev1.Span_SPAN_KIND_CLIENT)
				add("B", 2, 1, tracev1.Span_SPAN_KIND_SERVER)
			}
			s.Expire(time.Now().Add(2 * time.Hour))
			require.ElementsMatch(t, []string{"A->gateway", "gateway->B"}, pairs)
		})
	}
}

// Local internal children do not instrument an external peer. They must not
// suppress the virtual-server edge inferred from the client span's peer name.
func TestReviewLocalChildKeepsExternalPeerInference(t *testing.T) {
	for _, withChild := range []bool{false, true} {
		name := "without local child"
		if withChild {
			name = "with local child"
		}
		t.Run(name, func(t *testing.T) {
			expired := 0
			s := NewSpanStore(time.Hour, 10, func(*Edge) { t.Fatal("no remote server span was ingested") }, func(e *Edge) {
				expired++
				require.Equal(t, "A", e.ClientService)
				require.Equal(t, "external-B", e.PeerNode)
			}, nil)
			require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, 1), nil, tracev1.Span_SPAN_KIND_CLIENT, false, func(e *Edge, _ map[string]string) {
				e.ClientService = "A"
				e.SpanKind = tracev1.Span_SPAN_KIND_CLIENT
				e.PeerNode = "external-B"
			}))
			if withChild {
				require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, 2), reviewID(8, 1), tracev1.Span_SPAN_KIND_INTERNAL, false, func(e *Edge, _ map[string]string) {
					e.ClientService = "A"
					e.SpanKind = tracev1.Span_SPAN_KIND_INTERNAL
				}))
			}
			s.mu.Lock()
			s.expire(time.Now().Add(2 * time.Hour))
			s.mu.Unlock()
			require.Equal(t, 1, expired, "same-service internals must not hide an uninstrumented external destination")
		})
	}
}

func reviewID(size int, value byte) []byte {
	id := make([]byte, size)
	id[size-1] = value
	return id
}

// Every arrival order must resolve shared IDs before attaching local descendants
// or emitting metrics. A same-service descendant cannot steal the remote parent.
func TestSpanStoreSharedIDsWithDescendants(t *testing.T) {
	type span struct {
		service    string
		id, parent byte
		kind       tracev1.Span_SpanKind
	}
	nodes := []span{
		{"A", 1, 0, tracev1.Span_SPAN_KIND_INTERNAL},
		{"gateway", 2, 1, tracev1.Span_SPAN_KIND_CLIENT},
		{"B", 2, 1, tracev1.Span_SPAN_KIND_SERVER},
		{"B", 3, 2, tracev1.Span_SPAN_KIND_INTERNAL},
		{"C", 4, 3, tracev1.Span_SPAN_KIND_SERVER},
	}
	var visit func([]int, int)
	visit = func(order []int, pos int) {
		if pos < len(order) {
			for i := pos; i < len(order); i++ {
				order[pos], order[i] = order[i], order[pos]
				visit(order, pos+1)
				order[pos], order[i] = order[i], order[pos]
			}
			return
		}
		var pairs []string
		s := NewSpanStore(time.Hour, len(nodes), func(e *Edge) { pairs = append(pairs, e.ClientService+"->"+e.ServerService) }, func(*Edge) { t.Fatal("complete trace must not infer virtual edges") }, nil)
		for replay := 0; replay < 2; replay++ {
			for _, i := range order {
				n := nodes[i]
				var parent []byte
				if n.parent != 0 {
					parent = reviewID(8, n.parent)
				}
				require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, n.id), parent, n.kind, false, func(e *Edge, _ map[string]string) { e.ClientService = n.service; e.SpanKind = n.kind }))
			}
		}
		require.Empty(t, pairs, "edges must wait until shared endpoints settle")
		require.Len(t, s.spans, len(nodes), "replays must not consume additional cache entries")
		s.Expire(time.Now().Add(2 * time.Hour))
		require.ElementsMatch(t, []string{"A->gateway", "gateway->B", "B->C"}, pairs, "arrival order %v", order)
		require.Empty(t, s.spans)
		require.Empty(t, s.traces)
	}
	visit([]int{0, 1, 2, 3, 4}, 0)
}

// The wait budget belongs to each span, not the first span of a long trace.
// Expiring an unrelated old span must leave a recent client available to match.
func TestSpanStoreLongTraceKeepsRecentEndpoints(t *testing.T) {
	now := time.Now()
	var pairs []string
	s := NewSpanStore(10*time.Second, 10, func(e *Edge) { pairs = append(pairs, e.ClientService+"->"+e.ServerService) }, func(*Edge) {}, nil)
	s.now = func() time.Time { return now }
	add := func(id, parent byte, service string, kind tracev1.Span_SpanKind) {
		var parentID []byte
		if parent != 0 {
			parentID = reviewID(8, parent)
		}
		require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, id), parentID, kind, false, func(e *Edge, _ map[string]string) { e.ClientService = service; e.SpanKind = kind }))
	}
	add(1, 0, "unrelated", tracev1.Span_SPAN_KIND_INTERNAL)
	now = now.Add(9 * time.Second)
	add(2, 0, "A", tracev1.Span_SPAN_KIND_CLIENT)
	now = now.Add(time.Second)
	s.Expire(now)
	require.Len(t, s.spans, 1, "the recent client still has nine seconds left to match")
	now = now.Add(time.Second)
	add(3, 2, "B", tracev1.Span_SPAN_KIND_SERVER)
	s.Expire(now.Add(20 * time.Second))
	require.Equal(t, []string{"A->B"}, pairs)
}

// A late parent can outlive an intermediate span. Once the remote relationship
// is counted or filtered, retiring that intermediate must not infer it again.
func TestSpanStorePartialExpiryKeepsRemoteDecision(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		name := "remote observed"
		if filtered {
			name = "remote filtered"
		}
		t.Run(name, func(t *testing.T) {
			now := time.Now()
			completed, expired, dropped := 0, 0, 0
			s := NewSpanStore(10*time.Second, 10, func(*Edge) { completed++ }, func(*Edge) { expired++ }, func() { dropped++ })
			s.now = func() time.Time { return now }
			require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, 2), reviewID(8, 1), tracev1.Span_SPAN_KIND_INTERNAL, false, func(e *Edge, _ map[string]string) {
				e.ClientService = "A"
				e.SpanKind = tracev1.Span_SPAN_KIND_INTERNAL
			}))
			now = now.Add(time.Second)
			require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, 3), reviewID(8, 2), tracev1.Span_SPAN_KIND_SERVER, filtered, func(e *Edge, _ map[string]string) { e.ClientService = "B"; e.SpanKind = tracev1.Span_SPAN_KIND_SERVER }))
			now = now.Add(time.Second)
			require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, 1), nil, tracev1.Span_SPAN_KIND_CLIENT, false, func(e *Edge, _ map[string]string) {
				e.ClientService = "A"
				e.SpanKind = tracev1.Span_SPAN_KIND_CLIENT
				e.PeerNode = "B"
			}))
			now = now.Add(8 * time.Second)
			s.Expire(now)
			s.Expire(now.Add(time.Second))
			s.Expire(now.Add(2 * time.Second))
			if filtered {
				require.Zero(t, completed)
				require.Equal(t, 1, dropped)
			} else {
				require.Equal(t, 1, completed)
				require.Zero(t, dropped)
			}
			require.Zero(t, expired, "a retired intermediate must not cause a duplicate or filtered virtual relationship")
			require.Empty(t, s.spans)
		})
	}
}

// An observed database destination must suppress duplicate virtual inference
// even when its same-service parent remains cached longer than the database span.
func TestSpanStoreDatabaseDecisionSurvivesChildExpiry(t *testing.T) {
	now := time.Now()
	completed, expired := 0, 0
	s := NewSpanStore(10*time.Second, 10, func(e *Edge) { completed++; require.Equal(t, "postgres", e.ServerService) }, func(*Edge) { expired++ }, nil)
	s.now = func() time.Time { return now }
	require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, 2), reviewID(8, 1), tracev1.Span_SPAN_KIND_CLIENT, false, func(e *Edge, _ map[string]string) {
		e.ClientService = "A"
		e.ServerService = "postgres"
		e.ConnectionType = Database
		e.SpanKind = tracev1.Span_SPAN_KIND_CLIENT
	}))
	now = now.Add(time.Second)
	require.NoError(t, s.AddSpan(reviewID(16, 1), reviewID(8, 1), nil, tracev1.Span_SPAN_KIND_CLIENT, false, func(e *Edge, _ map[string]string) {
		e.ClientService = "A"
		e.PeerNode = "postgres"
		e.SpanKind = tracev1.Span_SPAN_KIND_CLIENT
	}))
	s.Expire(now.Add(9 * time.Second))
	s.Expire(now.Add(10 * time.Second))
	require.Equal(t, 1, completed)
	require.Zero(t, expired)
}
