package servicegraphs

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/generator/registry"
	filterconfig "github.com/grafana/tempo/v3/pkg/spanfilter/config"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

// TestRelaxedServiceGraphsBypassTopology exercises this tree through the public
// processor API. Numbers are span IDs; each span lasts its ID in seconds.
//
// A(1) -> gateway(2) -> proxy(3) -> B(4) -> sidecar(6) -> D(7)
//
//	|         |
//	|         +------> C(5)
//	+----------------> E(8)
//
// Each run interleaves two traces with identical span IDs, then replays both.
// Expected edges are explicit, rather than calculated by another tree walker.
func TestServiceGraphStoragePreservesTopology(t *testing.T) {
	nodes := []bypassTopologyNode{{"A", 0}, {"gateway", 1}, {"proxy", 2}, {"B", 3}, {"C", 2}, {"sidecar", 4}, {"D", 6}, {"E", 1}}
	raw := []bypassTopologyPair{{1, 2}, {2, 3}, {3, 4}, {2, 5}, {4, 6}, {6, 7}, {1, 8}}
	cases := []bypassTopologyCase{
		{name: "no presentation rules"},
		{name: "exact service hints", bypass: []string{"gateway", "proxy", "sidecar"}},
		{name: "anchored regex hints", patterns: []string{`^(gateway|proxy|sidecar)$`}},
		{name: "multiple regex hints", patterns: []string{`^gate[a-z]+$`, `^prox.$`, `^sidecar$`}},
		{name: "combined exact and regex hints", bypass: []string{"sidecar"}, patterns: []string{`^(gateway|proxy)$`}},
		{name: "partial service hints", patterns: []string{`^(gateway|sidecar)$`}},
		{name: "nonmatching anchored hint", patterns: []string{`^gate$`}},
		{name: "substring hint", patterns: []string{`gate`}},
		{name: "case sensitive hint", patterns: []string{`^GATEWAY$`}},
		{name: "span filter still applies independently", patterns: []string{`^(gateway|proxy|sidecar)$`}, excluded: "gateway", want: []bypassTopologyPair{{3, 4}, {4, 6}, {6, 7}, {1, 8}}},
		{name: "blacklisted root is retained", patterns: []string{`^(A|gateway|proxy|sidecar)$`}},
		{name: "all services blacklisted are retained", patterns: []string{`.*`}},
		{name: "overlapping hints do not duplicate metrics", bypass: []string{"gateway", "gateway"}, patterns: []string{`^gateway$`, `^(gateway|proxy|sidecar)$`}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.want == nil {
				tc.want = raw
			}
			for _, arrival := range []string{"forward", "reverse", "shuffle-0", "shuffle-42", "shuffle-2026"} {
				for _, batchSize := range []int{1, 5} {
					name := fmt.Sprintf("%s/batch-%d", arrival, batchSize)
					t.Run(name, func(t *testing.T) {
						runBypassTopology(t, nodes, tc, arrival, batchSize)
					})
				}
			}
		})
	}
}

type bypassTopologyNode struct {
	service string
	parent  int
}

type bypassTopologyPair struct{ client, server int }

type bypassTopologyCase struct {
	name     string
	bypass   []string
	patterns []string
	excluded string
	want     []bypassTopologyPair
}

func runBypassTopology(t *testing.T, nodes []bypassTopologyNode, tc bypassTopologyCase, arrival string, batchSize int) {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", nil)
	cfg.EnableServiceGraphEdgeStorage = true
	cfg.BypassServices = tc.bypass
	cfg.BypassServiceRegex = tc.patterns
	cfg.Workers = 0
	cfg.Wait = time.Hour
	cfg.MaxItems = 2 * len(nodes)
	cfg.EnableClientServerPrefix = true
	cfg.Dimensions = []string{"zone"}
	if tc.excluded != "" {
		cfg.FilterPolicies = []filterconfig.FilterPolicy{{Exclude: &filterconfig.PolicyMatch{
			MatchType: filterconfig.Strict, Attributes: []filterconfig.MatchPolicyAttribute{{Key: "resource.service.name", Value: tc.excluded}},
		}}}
	}
	metrics := registry.NewTestRegistry()
	p, err := New(cfg, "bypass-topology-test", metrics, log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
	require.NoError(t, err)
	t.Cleanup(func() { p.Shutdown(context.Background()) })
	var batches []*tracev1.ResourceSpans
	for i, n := range nodes {
		for trace := uint64(1); trace <= 2; trace++ {
			traceID := make([]byte, 16)
			binary.BigEndian.PutUint64(traceID[8:], trace)
			spanID := make([]byte, 8)
			binary.BigEndian.PutUint64(spanID, uint64(i+1))
			var parentID []byte
			if n.parent != 0 {
				parentID = make([]byte, 8)
				binary.BigEndian.PutUint64(parentID, uint64(n.parent))
			}
			kind := tracev1.Span_SPAN_KIND_INTERNAL
			batch := makeServiceGraphBatch(n.service, kind, traceID, spanID, parentID, tempopb.MakeKeyValueStringPtr("zone", fmt.Sprintf("%s-%d", n.service, trace)))
			span := batch.ScopeSpans[0].Spans[0]
			span.StartTimeUnixNano = uint64(time.Second)
			span.EndTimeUnixNano = span.StartTimeUnixNano + uint64(i+1)*uint64(time.Second)
			batches = append(batches, batch)
		}
	}
	switch arrival {
	case "reverse":
		slices.Reverse(batches)
	case "shuffle-0", "shuffle-42", "shuffle-2026":
		seed := map[string]int64{"shuffle-0": 0, "shuffle-42": 42, "shuffle-2026": 2026}[arrival]
		rand.New(rand.NewSource(seed)).Shuffle(len(batches), func(i, j int) { batches[i], batches[j] = batches[j], batches[i] })
	}
	// A replay must leave both counters and histograms unchanged.
	for replay := 0; replay < 2; replay++ {
		for start := 0; start < len(batches); start += batchSize {
			p.PushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: batches[start:min(start+batchSize, len(batches))]})
		}
		var wantRequests []string
		for _, edge := range tc.want {
			for trace := 1; trace <= 2; trace++ {
				client, server := nodes[edge.client-1].service, nodes[edge.server-1].service
				lbls := labels.FromMap(map[string]string{"client": client, "server": server, "client_zone": fmt.Sprintf("%s-%d", client, trace), "server_zone": fmt.Sprintf("%s-%d", server, trace)})
				wantRequests = append(wantRequests, fmt.Sprintf("%s%s 1", metricRequestTotal, lbls.String()))
				require.Equal(t, 1.0, metrics.Query(metricRequestTotal, lbls), "replay %d: trace %d: %s -> %s", replay, trace, client, server)
				require.Equal(t, 1.0, metrics.Query(metricRequestClientSeconds+"_count", lbls))
				require.Equal(t, 1.0, metrics.Query(metricRequestServerSeconds+"_count", lbls))
				require.Equal(t, float64(edge.client), metrics.Query(metricRequestClientSeconds+"_sum", lbls), "latency must come from visible client")
				require.Equal(t, float64(edge.server), metrics.Query(metricRequestServerSeconds+"_sum", lbls), "latency must come from visible server")
			}
		}
		var gotRequests []string
		for _, line := range strings.Split(metrics.String(), "\n") {
			if strings.HasPrefix(line, metricRequestTotal+"{") {
				gotRequests = append(gotRequests, line)
			}
		}
		require.ElementsMatch(t, wantRequests, gotRequests, "must emit exactly the expected edges, without hidden or cross-branch edges")
	}
}

func TestServiceGraphsBypassRegexValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		patterns  []string
		enabled   bool
		wantError string
	}{
		{name: "valid expressions", patterns: []string{`^gateway-[a-z]+$`, `^proxy/[0-9]+$`, `^sidecar\.mesh$`}, enabled: true},
		{name: "unclosed character class", patterns: []string{`[`}, enabled: true, wantError: "bypass_service_regex"},
		{name: "invalid expression after valid expression", patterns: []string{`^gateway$`, `(`}, enabled: true, wantError: "bypass_service_regex"},
		{name: "unsupported lookahead", patterns: []string{`gateway(?=east)`}, enabled: true, wantError: "bypass_service_regex"},
		{name: "presentation patterns do not enable storage", patterns: []string{`gateway`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			cfg.EnableServiceGraphEdgeStorage = tc.enabled
			cfg.BypassServiceRegex = tc.patterns
			cfg.Workers = 0
			p, err := New(cfg, "bypass-regex-validation", registry.NewTestRegistry(), log.NewNopLogger(), prometheus.NewCounter(prometheus.CounterOpts{}), prometheus.NewCounter(prometheus.CounterOpts{}))
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
			} else {
				require.NoError(t, err)
				t.Cleanup(func() { p.Shutdown(context.Background()) })
			}
		})
	}
}
