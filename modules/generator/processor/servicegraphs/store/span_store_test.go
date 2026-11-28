package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

func TestSpanStoreBranchingAndArrivalOrder(t *testing.T) {
	for _, order := range [][]byte{{1, 2, 3}, {1, 3, 2}, {2, 1, 3}, {2, 3, 1}, {3, 1, 2}, {3, 2, 1}} {
		t.Run(string([]byte{'0' + order[0], '0' + order[1], '0' + order[2]}), func(t *testing.T) {
			var pairs []string
			s := NewSpanStore(time.Hour, 3, func(e *Edge) { pairs = append(pairs, e.ClientService+"->"+e.ServerService) }, func(*Edge) {}, nil)
			for _, id := range order {
				var parent []byte
				if id != 1 {
					parent = []byte{1}
				}
				require.NoError(t, s.AddSpan([]byte{1}, []byte{id}, parent, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) {
					e.ClientService = string([]byte{'A' + id - 1})
					e.SpanMultiplier = 1
					e.SetTraceID([]byte{1})
				}))
			}
			require.Len(t, s.spans, 3)
			require.NoError(t, s.AddSpan([]byte{1}, []byte{2}, []byte{1}, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, nil))
			s.Expire(time.Now().Add(2 * time.Hour))
			require.ElementsMatch(t, []string{"A->B", "A->C"}, pairs)
			require.Len(t, pairs, 2, "duplicates within a window must not duplicate metrics")
		})
	}
}

func TestSpanStoreBoundsAndExpiry(t *testing.T) {
	s := NewSpanStore(time.Hour, 2, func(*Edge) {}, func(*Edge) {}, nil)
	add := func(id byte) error {
		return s.AddSpan([]byte{1}, []byte{id}, []byte{9}, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) { e.ClientService = "A" })
	}
	require.NoError(t, add(1))
	require.NoError(t, add(2))
	require.ErrorIs(t, add(3), ErrTooManyItems)
	require.Len(t, s.spans, 2)
	require.Len(t, s.traces, 1)
	s.mu.Lock()
	s.expire(time.Now().Add(2 * time.Hour))
	s.mu.Unlock()
	require.Empty(t, s.spans)
	require.Empty(t, s.traces, "expiration must release unresolved child references")
	require.Equal(t, 0, s.order.Len())
	require.NoError(t, add(3))
}

func TestSpanStoreFilteredSpansAndSameService(t *testing.T) {
	for _, tc := range []struct {
		name        string
		filteredID  byte
		sameService bool
	}{
		{name: "filtered parent", filteredID: 1},
		{name: "filtered child", filteredID: 2},
		{name: "same service", sameService: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			completed := 0
			dropped := 0
			s := NewSpanStore(time.Hour, 2, func(*Edge) { completed++ }, func(*Edge) {}, func() { dropped++ })
			for _, id := range []byte{2, 1} {
				var parent []byte
				if id == 2 {
					parent = []byte{1}
				}
				require.NoError(t, s.AddSpan([]byte{1}, []byte{id}, parent, tracev1.Span_SPAN_KIND_UNSPECIFIED, id == tc.filteredID, func(e *Edge, _ map[string]string) {
					e.ClientService = "A"
					if id == 2 && !tc.sameService {
						e.ClientService = "B"
					}
				}))
			}
			s.Expire(time.Now().Add(2 * time.Hour))
			require.Zero(t, completed)
			if tc.filteredID != 0 {
				require.Equal(t, 1, dropped)
			} else {
				require.Zero(t, dropped)
			}
			require.Empty(t, s.traces)
		})
	}
}

func TestSpanStoreConcurrentArrival(t *testing.T) {
	completed := 0
	s := NewSpanStore(time.Hour, 32, func(*Edge) { completed++ }, func(*Edge) {}, nil)
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for id := byte(1); id <= 32; id++ {
		wg.Add(1)
		go func(id byte) {
			defer wg.Done()
			var parent []byte
			if id != 1 {
				parent = []byte{1}
			}
			errs <- s.AddSpan([]byte{1}, []byte{id}, parent, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) { e.ClientService = string([]byte{id}) })
		}(id)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	s.Expire(time.Now().Add(2 * time.Hour))
	require.Equal(t, 31, completed)
	require.Empty(t, s.traces)
}

func TestSpanStoreExpiryVirtualSides(t *testing.T) {
	for _, kind := range []tracev1.Span_SpanKind{tracev1.Span_SPAN_KIND_SERVER, tracev1.Span_SPAN_KIND_CLIENT, tracev1.Span_SPAN_KIND_INTERNAL} {
		t.Run(kind.String(), func(t *testing.T) {
			expired := 0
			s := NewSpanStore(time.Hour, 1, func(*Edge) { t.Fatal("unpaired span completed") }, func(e *Edge) {
				expired++
				if kind == tracev1.Span_SPAN_KIND_SERVER {
					require.True(t, e.IsRoot())
					require.Equal(t, "A", e.ServerService)
					require.Empty(t, e.ClientService)
				} else {
					require.Equal(t, "A", e.ClientService)
				}
			}, nil)
			require.NoError(t, s.AddSpan([]byte{1}, []byte{1}, nil, kind, false, func(e *Edge, _ map[string]string) { e.SpanKind = kind; e.ClientService = "A" }))
			s.mu.Lock()
			s.expire(time.Now().Add(2 * time.Hour))
			s.mu.Unlock()
			if kind == tracev1.Span_SPAN_KIND_INTERNAL {
				require.Zero(t, expired)
			} else {
				require.Equal(t, 1, expired)
			}
		})
	}
}

func TestSpanStoreCopiesInputIDsAndSeparatesTraces(t *testing.T) {
	completed := 0
	s := NewSpanStore(time.Hour, 4, func(e *Edge) {
		completed++
		require.Equal(t, []byte{1}, e.TraceID())
	}, func(*Edge) {}, nil)
	traceID, spanID := []byte{1}, []byte{1}
	require.NoError(t, s.AddSpan(traceID, spanID, nil, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) {
		e.ClientService = "A"
		e.SetTraceID(traceID)
	}))
	traceID[0] = 9
	spanID[0] = 9
	// The same parent/span IDs in a different trace must not complete this edge.
	require.NoError(t, s.AddSpan([]byte{2}, []byte{2}, []byte{1}, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) { e.ClientService = "B" }))
	require.Zero(t, completed)
	require.NoError(t, s.AddSpan([]byte{1}, []byte{2}, []byte{1}, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) { e.ClientService = "B" }))
	s.Expire(time.Now().Add(2 * time.Hour))
	require.Equal(t, 1, completed)
}

func TestSpanStorePreservesIntermediateServices(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprint(filtered), func(t *testing.T) {
			var pairs []string
			expired := 0
			s := NewSpanStore(time.Hour, 4, func(e *Edge) { pairs = append(pairs, e.ClientService+"->"+e.ServerService) }, func(*Edge) { expired++ }, nil)
			for _, id := range []byte{3, 4, 2, 1} {
				service := map[byte]string{1: "A", 2: "gateway", 3: "B", 4: "C"}[id]
				var parent []byte
				if id == 2 {
					parent = []byte{1}
				} else if id > 2 {
					parent = []byte{2}
				}
				require.NoError(t, s.AddSpan([]byte{1}, []byte{id}, parent, tracev1.Span_SPAN_KIND_INTERNAL, filtered && id == 2, func(e *Edge, _ map[string]string) {
					e.ClientService = service
					e.SpanKind = tracev1.Span_SPAN_KIND_INTERNAL
				}))
			}
			s.Expire(time.Now().Add(2 * time.Hour))
			if filtered {
				require.Empty(t, pairs)
			} else {
				require.ElementsMatch(t, []string{"A->gateway", "gateway->B", "gateway->C"}, pairs)
			}
			require.Empty(t, s.traces)
			s.expire(time.Now().Add(2 * time.Hour))
			require.Zero(t, expired)
			require.Empty(t, s.spans)
			require.Empty(t, s.traces)
		})
	}
}

func TestSpanStoreMissingParentExpiry(t *testing.T) {
	s := NewSpanStore(time.Hour, 2, func(e *Edge) { require.Equal(t, "gateway", e.ClientService); require.Equal(t, "B", e.ServerService) }, func(*Edge) {}, nil)
	for _, id := range []byte{3, 2} {
		require.NoError(t, s.AddSpan([]byte{1}, []byte{id}, []byte{id - 1}, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) {
			e.ClientService = "B"
			if id == 2 {
				e.ClientService = "gateway"
			}
		}))
	}
	require.Len(t, s.traces, 1)
	s.expire(time.Now().Add(2 * time.Hour))
	require.Empty(t, s.traces)
	require.Empty(t, s.spans)
}

func BenchmarkSpanStoreFullTopology(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := NewSpanStore(time.Hour, 3, func(*Edge) {}, func(*Edge) {}, nil)
		for id := byte(1); id <= 3; id++ {
			var parent []byte
			if id > 1 {
				parent = []byte{id - 1}
			}
			if err := s.AddSpan([]byte{1}, []byte{id}, parent, tracev1.Span_SPAN_KIND_UNSPECIFIED, false, func(e *Edge, _ map[string]string) { e.ClientService = string([]byte{'A' + id - 1}) }); err != nil {
				b.Fatal(err)
			}
		}
		s.Expire(time.Now().Add(2 * time.Hour))
	}
}
