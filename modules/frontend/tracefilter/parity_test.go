package tracefilter

import (
	"encoding/binary"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

// TestFilterParityWithNormalSearch runs every parity query against the in-memory Filter, asserting
// equality with a real vp5 search where they agree and pinning the known divergences (diverges=true)
// where they do not. The Filter checks each event/link element (see protospan.go), so it gets the pure
// same-scope cases right - `event.a && event.b` needs one event with both, a dup key resolves per
// element - and matches a real search there.
//
// It diverges only where vp5 is not self-consistent or has a storage quirk we don't reproduce:
//   - under an OR, vp5 stops correlating a scope's conditions (allConditions flips off), so it can match
//     `(event.a && event.b) || z` on two different events - we still require one event with both.
//   - vp5 lets an unscoped `.k` reach an event/link only when a scoped `event.k` condition named that
//     same key - we resolve unscoped against span/resource/instrumentation only (no key tracking).
//   - `child.attr = nil` on a span whose events/links don't all carry attr, and vp5's storage quirks
//     (unscoped `.abc = nil` matching every span, array-member negation) which no evaluate path mirrors.
func TestFilterParityWithNormalSearch(t *testing.T) {
	trace, ids := makeParityTrace()
	block := buildVp5Block(t, trace) // build once, reused across every case below

	cases := []struct {
		name       string
		query      string
		structural bool
		diverges   bool
		// divergent rows pin both sides explicitly (wantSearch = vp5, wantFilter = ours) so a failure shows
		// which side changed. non-divergent rows leave these nil and just assert filter == vp5.
		wantSearch []string
		wantFilter []string
	}{
		{name: "matches equality on a span attribute", query: `{ .http.status_code = 500 }`},
		{name: "matches a resource attribute", query: `{ resource.service.name = "checkout" }`},
		{name: "matches a span-scoped attribute", query: `{ span.http.method = "GET" }`},
		{name: "matches the name intrinsic", query: `{ name = "db-query" }`},
		{name: "matches the kind intrinsic", query: `{ kind = client }`},
		{name: "matches the status intrinsic", query: `{ status = error }`},
		{name: "matches duration greater-than", query: `{ duration > 500ms }`},
		{name: "matches duration greater-or-equal", query: `{ duration >= 1s }`},
		{name: "matches a span attribute greater-or-equal", query: `{ .http.status_code >= 400 }`},
		{name: "matches an int attribute less-than", query: `{ .http.status_code < 500 }`},
		{name: "matches an int attribute greater-than", query: `{ .http.status_code > 450 }`},
		{name: "matches a string attribute by regex", query: `{ span.http.method =~ "G.*" }`},
		{name: "matches a member of a string array", query: `{ span.tags = "b" }`},
		{name: "no match on a string-array non-member", query: `{ span.tags = "zzz" }`},
		{name: "matches a member of an int array", query: `{ span.ports = 443 }`},
		{name: "single-element array collapses to scalar", query: `{ span.zone = "z1" }`},
		{name: "matches an event attribute", query: `{ event.exception.message = "boom" }`},
		{name: "matches the event:name intrinsic", query: `{ event:name = "cache-miss" }`},
		{name: "matches a link attribute", query: `{ link.rel = "child" }`},
		{name: "matches an instrumentation attribute", query: `{ instrumentation.telemetry.sdk = "otel" }`},
		{name: "matches the instrumentation:name intrinsic", query: `{ instrumentation:name = "authz-scope" }`},
		{name: "same-event AND", query: `{ event.exception.message = "boom" && event:name = "exception" }`},
		{name: "cross-event AND no match", query: `{ event.exception.message = "boom" && event.level = "error" }`},
		{name: "event dup key second value", query: `{ event.cache.key = "user:2" }`},
		{name: "cross-scope AND span+event", query: `{ .http.status_code = 500 && event.exception.message = "boom" }`},
		{name: "cross-scope AND event+link", query: `{ event.exception.message = "boom" && link.rel = "child" }`},
		{name: "cross-scope OR", query: `{ name = "auth" || event.exception.message = "boom" }`},
		{name: "cross-event AND under OR", query: `{ (event.exception.message = "boom" && event.level = "error") || name = "auth" }`, diverges: true, wantSearch: hexIDs(ids, "B", "D"), wantFilter: hexIDs(ids, "D")},
		{name: "unscoped alone does not reach event attr", query: `{ .exception.message = "boom" }`},
		{name: "unscoped reaches event attr when scope referenced", query: `{ event.exception.message = "boom" && .exception.message = "boom" }`, diverges: true, wantSearch: hexIDs(ids, "B"), wantFilter: nil},
		{name: "unscoped does not reach an unreferenced event key", query: `{ event.exception.message = "boom" && .level = "error" }`},
		{name: "attr = nil matches spans missing it", query: `{ span.abc = nil }`},
		{name: "attr != nil matches spans having it", query: `{ span.abc != nil }`},
		{name: "attr = nil matches nothing when all have it", query: `{ span.http.method = nil }`},
		{name: "attr != nil matches all when all have it", query: `{ span.http.method != nil }`},
		{name: "missing attr = nil combined with a match", query: `{ span.abc = nil && .http.status_code = 200 }`},
		{name: "structural descendant", query: `{ name = "root" } >> { .http.status_code = 500 }`, structural: true},

		{name: "negation span attr", query: `{ .http.status_code != 500 }`},
		{name: "negation intrinsic name", query: `{ name != "root" }`},
		{name: "negation intrinsic status", query: `{ status != error }`},
		{name: "negation intrinsic kind", query: `{ kind != client }`},
		{name: "negation event attr", query: `{ event.exception.message != "boom" }`},
		{name: "negation event attr other key", query: `{ event.level != "error" }`},
		{name: "negation event intrinsic name", query: `{ event:name != "exception" }`},
		{name: "negation link attr", query: `{ link.rel != "child" }`},
		{name: "negation instrumentation attr", query: `{ instrumentation.telemetry.sdk != "otel" }`},
		{name: "eq non-matching value", query: `{ .http.status_code = 400 }`},
		{name: "nonexistent span attr", query: `{ span.nonexistent = "x" }`},
		{name: "nonexistent event attr", query: `{ event.nonexistent = "x" }`},
		{name: "cross-scope AND event+link boolean", query: `{ event.exception.message = "boom" && link.rel = "child" }`},
		{name: "cross-event AND under OR boolean", query: `{ (event.exception.message = "boom" && event.level = "error") || name = "auth" }`, diverges: true, wantSearch: hexIDs(ids, "B", "D"), wantFilter: hexIDs(ids, "D")},
		{name: "cross-scope OR event/link", query: `{ event.exception.message = "boom" || link.rel = "child" }`},
		{name: "dup key single condition", query: `{ event.cache.key = "user:2" }`},
		{name: "dup key under OR", query: `{ event.cache.key = "user:2" || name = "auth" }`, diverges: true, wantSearch: hexIDs(ids, "D"), wantFilter: hexIDs(ids, "C", "D")},
		{name: "dup key negation", query: `{ event.cache.key != "user:1" }`},
		{name: "span exists", query: `{ span.abc = nil }`},
		{name: "span not-exists", query: `{ span.abc != nil }`},
		{name: "unscoped not-exists negation", query: `{ .abc != nil }`},
		{name: "event exists nil", query: `{ event.exception.message = nil }`, diverges: true, wantSearch: hexIDs(ids, "root", "B", "C"), wantFilter: hexIDs(ids, "root", "B", "C", "D")},
		{name: "event not-exists nil", query: `{ event.exception.message != nil }`},
		{name: "event dup key nil", query: `{ event.cache.key = nil }`, diverges: true, wantSearch: hexIDs(ids, "root", "B"), wantFilter: hexIDs(ids, "root", "B", "D")},
		{name: "link exists nil", query: `{ link.rel = nil }`, diverges: true, wantSearch: nil, wantFilter: hexIDs(ids, "root", "C", "D")},
		{name: "link not-exists nil", query: `{ link.rel != nil }`},
		{name: "instrumentation exists nil", query: `{ instrumentation.telemetry.sdk = nil }`},
		{name: "instrumentation not-exists nil", query: `{ instrumentation.telemetry.sdk != nil }`},
		// vp5 storage quirks no engine-evaluate path reproduces.
		{name: "unscoped not-exists quirk", query: `{ .abc = nil }`, diverges: true, wantSearch: hexIDs(ids, "root", "B", "C", "D"), wantFilter: hexIDs(ids, "root", "D")},
		{name: "array member negation string", query: `{ span.tags != "a" }`, diverges: true, wantSearch: hexIDs(ids, "D"), wantFilter: nil},
		{name: "array member negation int", query: `{ span.ports != 80 }`, diverges: true, wantSearch: hexIDs(ids, "D"), wantFilter: nil},
		{name: "array member negation regex", query: `{ span.tags !~ "a.*" }`, diverges: true, wantSearch: hexIDs(ids, "D"), wantFilter: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// ground truth: a real vp5 search over an in-memory block (see vp5_search_test.go).
			gt := searchVp5Block(t, block, tc.query)

			f, err := Options{Query: tc.query}.Compile()
			if tc.structural {
				require.Error(t, err, "CompileSpansetFilter rejects structural operators")
				return
			}
			require.NoError(t, err)

			out, err := f.Process(trace)
			require.NoError(t, err)
			got := matchedHexIDs(out)

			if tc.diverges {
				// pin both sides: a failure tells us whether vp5 or the filter changed, not just that they converged.
				require.Equal(t, tc.wantSearch, gt, "vp5 search output changed for %q", tc.query)
				require.Equal(t, tc.wantFilter, got, "filter output changed for %q", tc.query)
				require.NotEqual(t, tc.wantSearch, tc.wantFilter, "%q is marked divergent but the two expectations match", tc.query)
				return
			}
			require.Equal(t, gt, got, "the filter must match a real vp5 search for %q", tc.query)
		})
	}
}

// TestArrayNegationDivergesFromSearch pins a known, accepted divergence: negating an array-valued
// attribute against one of its members. vp5 pushes the predicate to the value columns and keeps the
// surviving (non-matching) elements, so the span matches. The in-memory filter evaluates the whole
// array with TraceQL's not-in semantics, so it does not. Positive array ops (=, =~, >, <) and negation
// of a non-member agree and are covered by the parity table. If a change makes these agree, move them
// into the parity table and delete this test.
func TestArrayNegationDivergesFromSearch(t *testing.T) {
	trace, _ := makeParityTrace() // D has tags=[a,b,c], ports=[80,443]
	block := buildVp5Block(t, trace)
	for _, q := range []string{`{ span.tags != "a" }`, `{ span.ports != 80 }`, `{ span.tags !~ "a.*" }`} {
		t.Run(q, func(t *testing.T) {
			gt := searchVp5Block(t, block, q)
			f, err := Options{Query: q}.Compile()
			require.NoError(t, err)
			out, err := f.Process(trace)
			require.NoError(t, err)
			require.NotEqual(t, gt, matchedHexIDs(out), "array-member negation is a documented divergence")
		})
	}
}

// TestFilterNilExistsMatrix explicitly checks `= nil` and `!= nil` against a real vp5 search over the
// full present/absent matrix: span.abc exists on B and C, and is absent on root and D. The exact-set
// assertions confirm each operator both includes the right spans and excludes the others.
func TestFilterNilExistsMatrix(t *testing.T) {
	trace, ids := makeParityTrace()
	block := buildVp5Block(t, trace)
	present := hexIDs(ids, "B", "C")   // spans that have span.abc
	absent := hexIDs(ids, "root", "D") // spans that do not

	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"= nil returns the absent spans and excludes the present ones", `{ span.abc = nil }`, absent},
		{"!= nil returns the present spans and excludes the absent ones", `{ span.abc != nil }`, present},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gt := searchVp5Block(t, block, tc.query)
			require.Equal(t, tc.want, gt, "ground-truth vp5 search disagrees with the present/absent matrix")
			f, err := Options{Query: tc.query}.Compile()
			require.NoError(t, err)
			out, err := f.Process(trace)
			require.NoError(t, err)
			require.Equal(t, tc.want, matchedHexIDs(out), "the filter must match vp5 for %q", tc.query)
		})
	}
}

// TestFilterUnscopedNilExistsMatrix covers unscoped `.abc = nil` / `.abc != nil` over the present/absent
// matrix (abc present on B,C, absent on root,D). Exists (`!= nil`) matches a real vp5 search. Not-exists
// (`= nil`) is a documented divergence: vp5 dispatches an unscoped attribute to span+resource and its
// not-exists there matches EVERY span (a storage quirk), while the in-memory filter returns only the
// spans actually missing abc. Both behaviors are pinned so a change to either is caught.
func TestFilterUnscopedNilExistsMatrix(t *testing.T) {
	trace, ids := makeParityTrace()
	block := buildVp5Block(t, trace)
	present := hexIDs(ids, "B", "C")   // spans that have abc
	absent := hexIDs(ids, "root", "D") // spans that do not
	all := hexIDs(ids, "root", "B", "C", "D")

	t.Run("!= nil matches vp5 (present in, absent out)", func(t *testing.T) {
		gt := searchVp5Block(t, block, `{ .abc != nil }`)
		require.Equal(t, present, gt, "vp5 unscoped exists returns the present spans")
		f, err := Options{Query: `{ .abc != nil }`}.Compile()
		require.NoError(t, err)
		out, err := f.Process(trace)
		require.NoError(t, err)
		require.Equal(t, present, matchedHexIDs(out), "the filter must match vp5 for unscoped != nil")
	})

	t.Run("= nil diverges: vp5 matches all, the filter matches only the absent", func(t *testing.T) {
		gt := searchVp5Block(t, block, `{ .abc = nil }`)
		require.Equal(t, all, gt, "vp5 unscoped not-exists quirk matches every span")
		f, err := Options{Query: `{ .abc = nil }`}.Compile()
		require.NoError(t, err)
		out, err := f.Process(trace)
		require.NoError(t, err)
		require.Equal(t, absent, matchedHexIDs(out), "the filter returns only spans actually missing abc")
	})
}

// makeParityTrace builds a small trace with varied resources, attributes, kinds, statuses and
// durations so the parity queries exercise distinct code paths. Returns the trace and a name->span-id
// map for expressing expectations.
func makeParityTrace() (*tempopb.Trace, map[string][]byte) {
	traceID := make([]byte, 16)
	binary.BigEndian.PutUint64(traceID[8:], 0xABCDEF)

	ids := map[string][]byte{
		"root": spanID(1),
		"B":    spanID(2),
		"C":    spanID(3),
		"D":    spanID(4),
	}

	base := uint64(1000 * time.Second)
	span := func(id []byte, parent []byte, name string, kind tracev1.Span_SpanKind, status tracev1.Status_StatusCode, statusCode int64, method string, durNanos uint64) *tracev1.Span {
		return &tracev1.Span{
			TraceId:           traceID,
			SpanId:            id,
			ParentSpanId:      parent,
			Name:              name,
			Kind:              kind,
			StartTimeUnixNano: base,
			EndTimeUnixNano:   base + durNanos,
			Status:            &tracev1.Status{Code: status},
			Attributes: []*commonv1.KeyValue{
				intAttr("http.status_code", statusCode),
				strAttr("http.method", method),
			},
		}
	}

	rootSpan := span(ids["root"], nil, "root", tracev1.Span_SPAN_KIND_SERVER, tracev1.Status_STATUS_CODE_OK, 200, "GET", uint64(time.Second))
	rootSpan.Events = []*tracev1.Span_Event{{Name: "start", TimeUnixNano: base + 1}}

	// B carries two DIFFERENT events (exception, log) and a link, to exercise per-event grouping: a
	// query ANDing attrs from different events must not match.
	bSpan := span(ids["B"], ids["root"], "db-query", tracev1.Span_SPAN_KIND_CLIENT, tracev1.Status_STATUS_CODE_ERROR, 500, "GET", uint64(2*time.Second))
	bSpan.Events = []*tracev1.Span_Event{
		{Name: "exception", TimeUnixNano: base + 1, Attributes: []*commonv1.KeyValue{strAttr("exception.message", "boom")}},
		{Name: "log", TimeUnixNano: base + 2, Attributes: []*commonv1.KeyValue{strAttr("level", "error")}},
	}
	bSpan.Links = []*tracev1.Span_Link{{TraceId: traceID, SpanId: spanID(99), Attributes: []*commonv1.KeyValue{strAttr("rel", "child")}}}
	// abc exists only on B and C, to test = nil / != nil (attribute present vs absent).
	bSpan.Attributes = append(bSpan.Attributes, strAttr("abc", "present"))

	cSpan := span(ids["C"], ids["root"], "cache-get", tracev1.Span_SPAN_KIND_CLIENT, tracev1.Status_STATUS_CODE_OK, 500, "POST", uint64(100*time.Millisecond))
	// a single-element array must collapse to a scalar (vp5 does this on read).
	cSpan.Attributes = append(cSpan.Attributes, strArrayAttr("zone", "z1"), strAttr("abc", "present"))
	// two events repeat the same attr key with different values, exercising grouped per-event resolution
	// (single-condition query) and flat-merge first-match (the OR variant in the boolean parity test).
	cSpan.Events = []*tracev1.Span_Event{
		{Name: "cache-miss", TimeUnixNano: base + 1, Attributes: []*commonv1.KeyValue{strAttr("cache.key", "user:1")}},
		{Name: "cache-miss", TimeUnixNano: base + 2, Attributes: []*commonv1.KeyValue{strAttr("cache.key", "user:2")}},
	}

	dSpan := span(ids["D"], ids["root"], "auth", tracev1.Span_SPAN_KIND_INTERNAL, tracev1.Status_STATUS_CODE_ERROR, 403, "GET", uint64(500*time.Millisecond))
	// array attrs to exercise array-membership matching (parity with vp5's homogeneous-array storage).
	dSpan.Attributes = append(dSpan.Attributes, strArrayAttr("tags", "a", "b", "c"), intArrayAttr("ports", 80, 443))

	checkoutScope := &commonv1.InstrumentationScope{Name: "parity-scope", Version: "1.0", Attributes: []*commonv1.KeyValue{strAttr("telemetry.sdk", "otel")}}
	checkout := &tracev1.ResourceSpans{
		Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{strAttr("service.name", "checkout")}},
		ScopeSpans: []*tracev1.ScopeSpans{{Scope: checkoutScope, Spans: []*tracev1.Span{rootSpan, bSpan, cSpan}}},
	}
	authzScope := &commonv1.InstrumentationScope{Name: "authz-scope", Version: "2.0"}
	authz := &tracev1.ResourceSpans{
		Resource:   &resourcev1.Resource{Attributes: []*commonv1.KeyValue{strAttr("service.name", "auth")}},
		ScopeSpans: []*tracev1.ScopeSpans{{Scope: authzScope, Spans: []*tracev1.Span{dSpan}}},
	}

	return &tempopb.Trace{ResourceSpans: []*tracev1.ResourceSpans{checkout, authz}}, ids
}

func spanID(n uint64) []byte {
	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, n)
	return id
}

// hexIDs returns the sorted hex span ids for the named spans, matching the engine's id spelling.
func hexIDs(ids map[string][]byte, names ...string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, util.SpanIDToHexString(ids[n]))
	}
	sort.Strings(out)
	return out
}
