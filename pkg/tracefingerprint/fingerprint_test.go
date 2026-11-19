package tracefingerprint

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/v3/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

type testSpan struct {
	id, parent    string
	service, name string
	kind          v1.Span_SpanKind
}

// newTrace puts every span in its own resource.
func newTrace(spans ...testSpan) *tempopb.Trace {
	tr := &tempopb.Trace{}
	for _, s := range spans {
		tr.ResourceSpans = append(tr.ResourceSpans, &v1.ResourceSpans{
			Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{serviceAttr(s.service)}},
			ScopeSpans: []*v1.ScopeSpans{{Spans: []*v1.Span{s.proto()}}},
		})
	}
	return tr
}

func (s testSpan) proto() *v1.Span {
	return &v1.Span{SpanId: []byte(s.id), ParentSpanId: []byte(s.parent), Name: s.name, Kind: s.kind}
}

// serviceAttr uses the literal OTLP key so the reference vectors also pin the
// key the package looks up.
func serviceAttr(service string) *commonv1.KeyValue {
	return &commonv1.KeyValue{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: service}}}
}

// referenceSpans builds the traces behind the contract's reference vectors.
func referenceSpans(scenario, name string) []testSpan {
	spans := []testSpan{
		{id: "a", service: "api", name: "GET /checkout"},
		{id: "b", parent: "a", service: "auth", name: "verify"},
		{id: "c", parent: "b", service: "db", name: name},
	}
	switch scenario {
	case "repeat":
		spans = append(spans, testSpan{id: "c2", parent: "c", service: "db", name: name})
	case "branch":
		spans = append(spans, testSpan{id: "d", parent: "a", service: "cache", name: "GET"})
	case "orphan":
		spans = []testSpan{spans[0], spans[2]}
	case "empty":
		spans[0].service, spans[0].name = "", ""
	}
	return spans
}

// TestReferenceVectors checks shingle hashes and fingerprints against the
// contract's reference vectors in testdata/reference-vectors.json. Each vector
// is checked twice, the second time with non-zero span kinds, which must not
// change the result.
func TestReferenceVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/reference-vectors.json")
	require.NoError(t, err)
	var vectors []struct {
		Scenario      string   `json:"scenario"`
		Count         uint32   `json:"count"`
		Signature     string   `json:"signature"`
		ShingleHashes []string `json:"shingleHashes"`
	}
	require.NoError(t, json.Unmarshal(raw, &vectors))
	require.Len(t, vectors, 7)

	kinds := []v1.Span_SpanKind{v1.Span_SPAN_KIND_SERVER, v1.Span_SPAN_KIND_CLIENT, v1.Span_SPAN_KIND_PRODUCER, v1.Span_SPAN_KIND_CONSUMER, v1.Span_SPAN_KIND_INTERNAL}
	var f Fingerprinter
	for _, v := range vectors {
		t.Run(v.Scenario, func(t *testing.T) {
			scenario, name := v.Scenario, "SELECT"
			switch scenario {
			case "unicode":
				scenario, name = "chain", "sélect 🦊"
			case "long":
				scenario, name = "chain", strings.Repeat("operation-", 30)
			}
			want, err := hex.DecodeString(strings.ReplaceAll(v.Signature, " ", ""))
			require.NoError(t, err)

			spans := referenceSpans(scenario, name)
			withKinds := slices.Clone(spans)
			for i := range withKinds {
				withKinds[i].kind = kinds[i%len(kinds)]
			}
			for _, tr := range []*tempopb.Trace{newTrace(spans...), newTrace(withKinds...)} {
				hashes, ok := f.shingleHashes(tr)
				require.True(t, ok)
				got := make([]string, len(hashes))
				for i, h := range hashes {
					got[i] = fmt.Sprintf("%016x", h)
				}
				require.Equal(t, v.ShingleHashes, got)

				res := f.Fingerprint(tr)
				require.Equal(t, StatusComputed, res.Status)
				require.Equal(t, "v1", res.Version)
				require.Equal(t, v.Count, res.ShingleCount)
				require.Equal(t, want, res.Fingerprint[:])
			}
		})
	}
}

// TestPathCases checks shingle counts and similarity scores against the
// contract's worked examples, where X is api "GET /checkout", Y is auth
// "verify" and Z is db "SELECT".
func TestPathCases(t *testing.T) {
	x := testSpan{id: "a", service: "api", name: "GET /checkout"}
	y := testSpan{id: "b", parent: "a", service: "auth", name: "verify"}
	z := testSpan{id: "c", parent: "b", service: "db", name: "SELECT"}
	j := testSpan{id: "j", service: "policy", name: "check"}
	k := testSpan{id: "k", parent: "j", service: "router", name: "route"}
	under := func(s testSpan, parent string) testSpan { s.parent = parent; return s }
	renamed := func(s testSpan, name string) testSpan { s.name = name; return s }

	var f Fingerprinter
	base := f.Fingerprint(newTrace(x, y, z))

	for _, tc := range []struct {
		name       string
		spans      []testSpan
		count      uint32
		similarity float64
	}{
		{"X→Y→Z", []testSpan{x, y, z}, 6, 1},
		{"X→Y→Z→Z", []testSpan{x, y, z, {id: "c2", parent: "c", service: "db", name: "SELECT"}}, 6, 1},
		{"X→Y→Y→Z", []testSpan{x, y, {id: "b2", parent: "b", service: "auth", name: "verify"}, under(z, "b2")}, 6, 1},
		{"X→Y→J→Z", []testSpan{x, y, under(j, "b"), under(z, "j")}, 9, 0.35546875},
		{"X→Y→J→K→Z", []testSpan{x, y, under(j, "b"), k, under(z, "k")}, 12, 0.2734375},
		{"X→Z", []testSpan{x, under(z, "a")}, 3, 0.25390625},
		{"X→{Y,Z}", []testSpan{x, y, under(z, "a")}, 5, 0.5859375},
		{"J→X→Y→Z", []testSpan{j, under(x, "j"), y, z}, 9, 0.703125},
		{"X→Y→Z'", []testSpan{x, y, renamed(z, "UPDATE")}, 6, 0.328125},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := f.Fingerprint(newTrace(tc.spans...))
			require.Equal(t, StatusComputed, res.Status)
			require.Equal(t, tc.count, res.ShingleCount)

			sim, err := Similarity(&base, &res)
			require.NoError(t, err)
			require.Equal(t, tc.similarity, sim)
			if tc.similarity == 1 {
				require.Equal(t, base, res)
			}
			reverse, err := Similarity(&res, &base)
			require.NoError(t, err)
			require.Equal(t, sim, reverse)
		})
	}
}

// TestDistinctShingles checks that a shingle seen on several paths counts
// once: A with two B children has the shingles A, B and A→B.
func TestDistinctShingles(t *testing.T) {
	a := testSpan{id: "a", service: "api", name: "GET"}
	b1 := testSpan{id: "b1", parent: "a", service: "db", name: "SELECT"}
	b2 := testSpan{id: "b2", parent: "a", service: "db", name: "SELECT"}

	var f Fingerprinter
	got := f.Fingerprint(newTrace(a, b1, b2))
	require.Equal(t, uint32(3), got.ShingleCount)
	require.Equal(t, f.Fingerprint(newTrace(a, b1)), got)
}

// TestRevisitedIdentity checks that only consecutive repeats collapse: in
// A→B→A the second A still adds B→A and A→B→A, for five distinct shingles.
func TestRevisitedIdentity(t *testing.T) {
	a := testSpan{id: "a", service: "api", name: "GET"}
	b := testSpan{id: "b", parent: "a", service: "db", name: "SELECT"}
	a2 := testSpan{id: "a2", parent: "b", service: "api", name: "GET"}

	var f Fingerprinter
	res := f.Fingerprint(newTrace(a, b, a2))
	require.Equal(t, StatusComputed, res.Status)
	require.Equal(t, uint32(5), res.ShingleCount)
}

func TestEmptyAndUnavailable(t *testing.T) {
	root := testSpan{id: "r", service: "api", name: "GET"}
	for _, tc := range []struct {
		name string
		tr   *tempopb.Trace
		want Status
	}{
		{"nil trace", nil, StatusEmpty},
		{"no spans", &tempopb.Trace{ResourceSpans: []*v1.ResourceSpans{{ScopeSpans: []*v1.ScopeSpans{{}}}}}, StatusEmpty},
		{"self parent", newTrace(testSpan{id: "a", parent: "a", service: "api", name: "GET"}), StatusUnavailable},
		{"two-span cycle", newTrace(root, testSpan{id: "a", parent: "b", name: "x"}, testSpan{id: "b", parent: "a", name: "y"}), StatusUnavailable},
		{"chain into cycle", newTrace(root, testSpan{id: "a", parent: "b"}, testSpan{id: "b", parent: "a"}, testSpan{id: "c", parent: "a"}), StatusUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var f Fingerprinter
			res := f.Fingerprint(tc.tr)
			require.Equal(t, Result{Version: Version, Status: tc.want}, res)
		})
	}
}

func TestSingleSpan(t *testing.T) {
	var f Fingerprinter
	res := f.Fingerprint(newTrace(testSpan{id: "a", service: "api", name: "GET"}))
	require.Equal(t, StatusComputed, res.Status)
	require.Equal(t, uint32(1), res.ShingleCount)
}

func TestInputOrderAndBatching(t *testing.T) {
	spans := []testSpan{
		{id: "a", service: "api", name: "GET /checkout"},
		{id: "b", parent: "a", service: "auth", name: "verify"},
		{id: "c", parent: "b", service: "db", name: "SELECT"},
		{id: "d", parent: "a", service: "cache", name: "GET"},
		{id: "e", parent: "missing", service: "db", name: "SELECT"},
	}
	// Same span ID and kind as b, different name. The smaller copy, refresh,
	// wins wherever either copy appears.
	conflict := testSpan{id: "b", parent: "a", service: "auth", name: "refresh"}

	reversed := make([]testSpan, 0, len(spans)+1)
	for i := len(spans) - 1; i >= 0; i-- {
		reversed = append(reversed, spans[i])
	}
	reversed = append(reversed, conflict)

	byService := &tempopb.Trace{}
	for _, s := range append([]testSpan{conflict}, spans...) {
		var rs *v1.ResourceSpans
		for _, existing := range byService.ResourceSpans {
			if existing.Resource.Attributes[0].Value.GetStringValue() == s.service {
				rs = existing
			}
		}
		if rs == nil {
			rs = &v1.ResourceSpans{Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{serviceAttr(s.service)}}, ScopeSpans: []*v1.ScopeSpans{{}, {}}}
			byService.ResourceSpans = append(byService.ResourceSpans, rs)
		}
		scope := rs.ScopeSpans[len(byService.ResourceSpans)%2]
		scope.Spans = append(scope.Spans, s.proto())
	}

	var f Fingerprinter
	want := f.Fingerprint(newTrace(spans[0], conflict, spans[2], spans[3], spans[4]))
	require.Equal(t, StatusComputed, want.Status)
	require.Equal(t, want, f.Fingerprint(newTrace(append(spans, conflict)...)))
	require.Equal(t, want, f.Fingerprint(newTrace(reversed...)))
	// A larger trace in between must not change later results.
	f.Fingerprint(syntheticTrace(1000, 4))
	require.Equal(t, want, f.Fingerprint(byService))
}

func TestDuplicateWinner(t *testing.T) {
	root := testSpan{id: "r", service: "frontend", name: "GET /"}
	for _, tc := range []struct {
		name          string
		winner, loser testSpan
	}{
		{
			name:   "same service, smaller name wins",
			winner: testSpan{id: "s", parent: "r", service: "auth", name: "refresh"},
			loser:  testSpan{id: "s", parent: "r", service: "auth", name: "verify"},
		},
		{
			name:   "service decides before name",
			winner: testSpan{id: "s", parent: "r", service: "auth", name: "zzz"},
			loser:  testSpan{id: "s", parent: "r", service: "billing", name: "aaa"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var f Fingerprinter
			want := f.Fingerprint(newTrace(root, tc.winner))
			require.Equal(t, want, f.Fingerprint(newTrace(root, tc.winner, tc.loser)))
			require.Equal(t, want, f.Fingerprint(newTrace(root, tc.loser, tc.winner)))
		})
	}
}

func TestServiceName(t *testing.T) {
	str := func(s string) *commonv1.AnyValue {
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: s}}
	}
	num := &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 7}}
	arr := &commonv1.AnyValue{Value: &commonv1.AnyValue_ArrayValue{ArrayValue: &commonv1.ArrayValue{Values: []*commonv1.AnyValue{str("cart")}}}}
	withAttrs := func(attrs ...*commonv1.KeyValue) *tempopb.Trace {
		tr := newTrace(testSpan{id: "a", name: "GET"})
		tr.ResourceSpans[0].Resource.Attributes = attrs
		return tr
	}

	for _, tc := range []struct {
		name    string
		tr      *tempopb.Trace
		service string
	}{
		{"later string wins", withAttrs(&commonv1.KeyValue{Key: attrServiceName, Value: str("a")}, &commonv1.KeyValue{Key: attrServiceName, Value: str("b")}), "b"},
		{"trailing non-string ignored", withAttrs(&commonv1.KeyValue{Key: attrServiceName, Value: str("checkout")}, &commonv1.KeyValue{Key: attrServiceName, Value: num}), "checkout"},
		{"leading non-string ignored", withAttrs(&commonv1.KeyValue{Key: attrServiceName, Value: num}, &commonv1.KeyValue{Key: attrServiceName, Value: str("checkout")}), "checkout"},
		{"only non-string", withAttrs(&commonv1.KeyValue{Key: attrServiceName, Value: num}), ""},
		{"one-element array ignored", withAttrs(&commonv1.KeyValue{Key: attrServiceName, Value: arr}), ""},
		{"nil string wrapper ignored", withAttrs(&commonv1.KeyValue{Key: attrServiceName, Value: str("checkout")}, &commonv1.KeyValue{Key: attrServiceName, Value: &commonv1.AnyValue{Value: (*commonv1.AnyValue_StringValue)(nil)}}), "checkout"},
		{"no resource", &tempopb.Trace{ResourceSpans: []*v1.ResourceSpans{{ScopeSpans: []*v1.ScopeSpans{{Spans: []*v1.Span{{SpanId: []byte("a"), Name: "GET"}}}}}}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var f Fingerprinter
			require.Equal(t, f.Fingerprint(newTrace(testSpan{id: "a", service: tc.service, name: "GET"})), f.Fingerprint(tc.tr))
		})
	}
}

func TestSpanReconciliation(t *testing.T) {
	root := testSpan{id: "r", service: "frontend", name: "GET /"}
	client := testSpan{id: "s", parent: "r", service: "frontend", name: "call", kind: v1.Span_SPAN_KIND_CLIENT}
	server := testSpan{id: "s", parent: "r", service: "backend", name: "handle", kind: v1.Span_SPAN_KIND_SERVER}
	internal := testSpan{id: "s", parent: "r", service: "backend", name: "work", kind: v1.Span_SPAN_KIND_INTERNAL}
	child := testSpan{id: "t", parent: "s", service: "backend", name: "query"}
	withID := func(s testSpan, id, parent string) testSpan { s.id, s.parent = id, parent; return s }
	zeroID := string(make([]byte, 8))

	for _, tc := range []struct {
		name       string
		got, equal []testSpan
	}{
		{
			name:  "replicated span",
			got:   []testSpan{root, child, client, client},
			equal: []testSpan{root, client, child},
		},
		{
			name:  "shared span ID siblings",
			got:   []testSpan{root, client, server, child},
			equal: []testSpan{root, withID(client, "c", "r"), withID(server, "v", "r"), withID(child, "t", "v")},
		},
		{
			name:  "server parented by its client",
			got:   []testSpan{root, client, withID(server, "s", "s"), child},
			equal: []testSpan{root, withID(client, "c", "r"), withID(server, "v", "c"), withID(child, "t", "v")},
		},
		{
			name:  "shared pair without the wanted kind",
			got:   []testSpan{root, client, internal, child},
			equal: []testSpan{root, withID(client, "c", "r"), withID(internal, "i", "r"), withID(child, "t", "")},
		},
		{
			name:  "three spans share the parent ID",
			got:   []testSpan{root, client, server, internal, child},
			equal: []testSpan{root, withID(client, "c", "r"), withID(server, "v", "r"), withID(internal, "i", "r"), withID(child, "t", "")},
		},
		{
			name:  "all-zero parent ID is a root",
			got:   []testSpan{root, withID(server, zeroID, ""), withID(child, "t", zeroID)},
			equal: []testSpan{root, withID(server, "v", ""), withID(child, "t", "")},
		},
		{
			name:  "same ID and kind with different parents",
			got:   []testSpan{root, withID(server, "v", ""), withID(child, "t", "v"), withID(child, "t", "r")},
			equal: []testSpan{root, withID(server, "v", ""), withID(child, "t", "r")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var f Fingerprinter
			got := f.Fingerprint(newTrace(tc.got...))
			require.Equal(t, StatusComputed, got.Status)
			require.Equal(t, f.Fingerprint(newTrace(tc.equal...)), got)
		})
	}
}

func TestSimilarityUnavailable(t *testing.T) {
	var f Fingerprinter
	computed := f.Fingerprint(newTrace(referenceSpans("chain", "SELECT")...))
	empty := f.Fingerprint(nil)
	otherVersion := computed
	otherVersion.Version = "v2"

	for _, tc := range []struct {
		name string
		a, b *Result
		err  error
	}{
		{"different versions", &computed, &otherVersion, ErrVersionMismatch},
		{"missing version", &computed, &Result{}, ErrVersionMismatch},
		{"unsupported version", &otherVersion, &otherVersion, ErrUnsupportedVersion},
		{"empty", &computed, &empty, ErrNoFingerprint},
		{"unavailable", &Result{Version: Version}, &computed, ErrNoFingerprint},
		{"nil first", nil, &computed, ErrNoFingerprint},
		{"nil second", &computed, nil, ErrNoFingerprint},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Similarity(tc.a, tc.b)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

// TestSimilarityFloor checks that fingerprints differing in more than half
// their bits score 0 without an error.
func TestSimilarityFloor(t *testing.T) {
	a := Result{Version: Version, Status: StatusComputed, ShingleCount: 1}
	b := a
	for i := range b.Fingerprint {
		b.Fingerprint[i] = 0xff
	}
	sim, err := Similarity(&a, &b)
	require.NoError(t, err)
	require.Equal(t, 0.0, sim)
}

// syntheticTrace builds a tree of n spans over eight services and 211 span
// names, so the number of distinct shingles keeps growing with n. Each span
// has up to fanout children: 1 builds a chain, n a star.
func syntheticTrace(n, fanout int) *tempopb.Trace {
	services := []string{"frontend", "cart", "checkout", "payment", "shipping", "email", "currency", "ads"}
	tr := &tempopb.Trace{}
	for _, s := range services {
		tr.ResourceSpans = append(tr.ResourceSpans, &v1.ResourceSpans{
			Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{serviceAttr(s)}},
			ScopeSpans: []*v1.ScopeSpans{{}},
		})
	}
	for i := range n {
		span := &v1.Span{
			SpanId: binary.BigEndian.AppendUint64(nil, uint64(i+1)),
			Name:   fmt.Sprintf("op-%d", i%211),
		}
		if i > 0 {
			span.ParentSpanId = binary.BigEndian.AppendUint64(nil, uint64((i-1)/fanout+1))
		}
		scope := tr.ResourceSpans[i%len(services)].ScopeSpans[0]
		scope.Spans = append(scope.Spans, span)
	}
	return tr
}

var benchmarkSizes = []int{10, 100, 1000, 10000}

func BenchmarkShingles(b *testing.B) {
	for _, n := range benchmarkSizes {
		b.Run(fmt.Sprintf("spans=%d", n), func(b *testing.B) {
			tr := syntheticTrace(n, 4)
			var f Fingerprinter
			b.ReportAllocs()
			for b.Loop() {
				if _, ok := f.shingleHashes(tr); !ok {
					b.Fatal("unavailable")
				}
			}
		})
	}
}

// BenchmarkShinglesShape compares a deep and a wide trace of the same size.
func BenchmarkShinglesShape(b *testing.B) {
	const n = 10000
	for _, shape := range []struct {
		name   string
		fanout int
	}{{"chain", 1}, {"star", n}} {
		b.Run(shape.name, func(b *testing.B) {
			tr := syntheticTrace(n, shape.fanout)
			var f Fingerprinter
			b.ReportAllocs()
			for b.Loop() {
				if _, ok := f.shingleHashes(tr); !ok {
					b.Fatal("unavailable")
				}
			}
		})
	}
}

// sketchOneAtATime is sketch without the four-shingle passes.
func sketchOneAtATime(shingles []uint64, fp *[Size]byte) {
	var mins [slots]uint64
	for i := range mins {
		mins[i] = math.MaxUint64
	}
	for _, x := range shingles {
		x = mix64(x)
		for i := range mins {
			mins[i] = min(mins[i], x*coefficientA[i]+coefficientB[i])
		}
	}
	for i, m := range mins {
		fp[i/8] |= byte(mix64(m)&1) << (i % 8)
	}
}

func TestSketchMatchesOneAtATime(t *testing.T) {
	// 0 to 9 shingles cover every remainder after the four-shingle passes.
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 1001} {
		var shingles []uint64
		for i := range n {
			shingles = append(shingles, mix64(uint64(i)))
		}
		var got, want [Size]byte
		sketch(shingles, &got)
		sketchOneAtATime(shingles, &want)
		require.Equal(t, want, got, "%d shingles", n)
	}
}

func BenchmarkSketch(b *testing.B) {
	for _, n := range benchmarkSizes {
		var f Fingerprinter
		hashes, _ := f.shingleHashes(syntheticTrace(n, 4))
		for _, impl := range []struct {
			name   string
			sketch func([]uint64, *[Size]byte)
		}{{"four-at-a-time", sketch}, {"one-at-a-time", sketchOneAtATime}} {
			b.Run(fmt.Sprintf("spans=%d/%s", n, impl.name), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var fp [Size]byte
					impl.sketch(hashes, &fp)
				}
				b.ReportMetric(float64(len(hashes)), "shingles")
			})
		}
	}
}

func BenchmarkFingerprint(b *testing.B) {
	for _, n := range benchmarkSizes {
		b.Run(fmt.Sprintf("spans=%d", n), func(b *testing.B) {
			tr := syntheticTrace(n, 4)
			var f Fingerprinter
			b.ReportAllocs()
			for b.Loop() {
				if f.Fingerprint(tr).Status != StatusComputed {
					b.Fatal("not computed")
				}
			}
		})
	}
}

func BenchmarkSimilarity(b *testing.B) {
	var f Fingerprinter
	x := f.Fingerprint(syntheticTrace(100, 4))
	y := f.Fingerprint(syntheticTrace(120, 4))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Similarity(&x, &y); err != nil {
			b.Fatal(err)
		}
	}
}
