// Package bloomgateway (query.go) implements tempopb.BloomGatewayServer --
// the read path (DESIGN.md § Query path, "Gateway-side execution" steps
// 1-5): resolve a trace ID to its leaf and fingerprint, then turn a leaf
// lookup plus a tenant-window subtraction into a rejection set.
//
// Server reads Group-B structures (the directory, registry, tenant set)
// only -- it has no dependency on the write path (events.go/consumer.go/
// worker.go) at all, matching the implementation plan's Wave-2 parallel-
// safety note for this WP.
package bloomgateway

import (
	"context"
	"sort"
	"time"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/grafana/tempo/pkg/deltauuid"
	"github.com/grafana/tempo/pkg/tempopb"
)

// wireVersion is BloomGatewayQueryResponse.Version's only currently-defined
// value (implementation plan §2).
const wireVersion uint32 = 1

// FlagLeafUnavailable marks a response as an abstention: the leaf addressed
// by the request is nil (unowned) or constructing (backfilling -- not yet
// safe to serve from, DESIGN.md § Design constraints: "a leaf is never
// served from partial state"), never a computation over real state. Bits
// 1-31 are reserved and always 0 in this version (§2).
const FlagLeafUnavailable uint32 = 1 << 0

// Query result labels for the queriesTotal{result} counter (DESIGN.md §
// Metrics: `queries_total{result="reject_all|candidates|empty"}`). The
// design doc names the three values but does not pin down their exact
// boundaries; this is the interpretation implemented and tested here:
//
//   - resultEmpty: the leaf was unavailable (FlagLeafUnavailable set) --
//     no computation happened at all, distinct from a computed-but-empty
//     candidate set below.
//   - resultRejectAll: the leaf answered and zero handles matched within
//     the tenant's window -- every block in the window is rejected
//     (DESIGN.md § Query path: "No match -> rejection set = A_X_window").
//   - resultCandidates: the leaf answered and at least one handle matched
//     within the window -- a genuine, non-empty candidate set.
const (
	resultEmpty      = "empty"
	resultRejectAll  = "reject_all"
	resultCandidates = "candidates"
)

// Server implements tempopb.BloomGatewayServer (DESIGN.md § Query path).
type Server struct {
	dir     *Directory
	reg     *Registry
	tenants *TenantSet

	// hashSeed and seedFingerprint are both derived ONCE from the raw
	// configured seed, at construction (AMENDMENT A4's domain separation,
	// hash.go) -- re-deriving either per query would double the hashing
	// cost of the hot query path for a value that never changes between
	// queries.
	hashSeed        uint64
	seedFingerprint uint64
	d, f            uint8

	metrics *metrics
}

var _ tempopb.BloomGatewayServer = (*Server)(nil)

// NewServer builds a Server. seed is the raw configured secret
// (bloomgateway.Config.Seed's plaintext) -- NewServer derives hashSeed and
// seedFingerprint from it once here, per hash.go's documented convention;
// callers must NOT pre-derive either themselves.
func NewServer(dir *Directory, reg *Registry, tenants *TenantSet, seed []byte, d, f uint8, m *metrics) *Server {
	return &Server{
		dir:             dir,
		reg:             reg,
		tenants:         tenants,
		hashSeed:        HashSeed(seed),
		seedFingerprint: SeedFingerprint(seed),
		d:               d,
		f:               f,
		metrics:         m,
	}
}

// Query implements tempopb.BloomGatewayServer (DESIGN.md § Query path,
// "Gateway-side execution"):
//
//  1. hash.go's Address derives (leafIdx, fingerprint) from the request's
//     raw trace ID under THIS instance's own configured seed/d/f -- the
//     client's own hash, if any, is used only for routing and is never
//     trusted for the actual answer (§ Protocol: "This makes seed skew
//     between QF and gateway degrade to fallback... instead of corrupting
//     answers"). Deviation from the plan's sketch, noted here because it
//     removes a step the plan called for: WP5's actual, already-landed
//     Address signature canonicalizes traceID via
//     util.PadTraceIDTo16Bytes internally and never returns an error, so
//     there is no separate "a padding error is a client error" path to
//     implement -- the plan described a padding failure mode that WP5's
//     real signature does not expose.
//  2. dir.Lookup resolves the leaf. ok=false (nil or constructing) means
//     unavailable: respond with FlagLeafUnavailable and no rejected set --
//     the safe-fallback case, never a computation (invariant #1, #7, §7).
//  3. tenants.Window scopes A_T to the request's tenant and time range (0
//     == unbounded on either side, matching the proto's own convention,
//     §2).
//  4. candidates := matched handles ∩ window; rejected := window minus
//     candidates. This single expression covers both the "no match" case
//     (candidates empty, rejected == window in full) and the "match" case
//     with no special-casing (invariant #2, §7): a block absent from
//     window can be neither a candidate nor ever appear in rejected,
//     regardless of whether it's simply unknown to this instance or still
//     Pending (TestQuery_RejectionRequiresLiveAndAT).
//  5. Resolve rejected's handles to UUIDs via the registry's batch
//     accessor under one lock (AMENDMENT A5 -- a per-handle
//     Registry.LookupHandle call for what can be an ~100k-handle unscoped
//     rejection set would be a needless hot-path cost), sort ascending,
//     and delta-encode (pkg/deltauuid).
func (s *Server) Query(_ context.Context, req *tempopb.BloomGatewayQueryRequest) (*tempopb.BloomGatewayQueryResponse, error) {
	queryStart := time.Now()

	leafIdx, fp := Address(req.TraceId, s.hashSeed, s.d, s.f)

	// Directory.Lookup's fp parameter is uint16 (leaf.go's v1 storage
	// width, WP9): safe to narrow here because Config.Validate rejects any
	// F > maxFingerprintBits (16, config.go) before this Server can ever be
	// constructed with a wider one -- Address itself only ever produces a
	// value that fits f bits, per its own contract.
	handles, ok := s.dir.Lookup(leafIdx, uint16(fp))
	if !ok {
		s.metrics.queriesTotal.WithLabelValues(resultEmpty).Inc()
		s.metrics.queryDurationSeconds.Observe(time.Since(queryStart).Seconds())
		return &tempopb.BloomGatewayQueryResponse{
			Version:         wireVersion,
			Flags:           FlagLeafUnavailable,
			SeedFingerprint: s.seedFingerprint,
		}, nil
	}

	window := s.tenants.Window(req.TenantId, unixNanoToTime(req.StartTimeUnixNano), unixNanoToTime(req.EndTimeUnixNano))

	matched := roaring.New()
	for _, h := range handles {
		matched.Add(uint32(h))
	}

	// Both And and AndNot below are the package-level functions (not the
	// mutating *Bitmap methods of the same name): they build a fresh
	// result bitmap without touching either operand's own containers
	// (verified against the vendored implementation) -- window is read
	// twice here and must survive both calls unmodified.
	candidates := roaring.And(window, matched)
	rejected := roaring.AndNot(window, candidates)

	encoded := s.encodeRejected(rejected)

	result := resultCandidates
	if candidates.IsEmpty() {
		result = resultRejectAll
	}
	s.metrics.queriesTotal.WithLabelValues(result).Inc()
	s.metrics.queryCandidates.Observe(float64(candidates.GetCardinality()))
	s.metrics.responseBytes.Observe(float64(len(encoded)))
	s.metrics.queryDurationSeconds.Observe(time.Since(queryStart).Seconds())

	return &tempopb.BloomGatewayQueryResponse{
		Version:         wireVersion,
		SeedFingerprint: s.seedFingerprint,
		Rejected:        encoded,
	}, nil
}

// encodeRejected resolves rejected's handles to UUIDs via the registry's
// batch accessor (AMENDMENT A5), sorts them ascending, and delta-encodes
// them (pkg/deltauuid) -- BloomGatewayQueryResponse.Rejected's wire format
// (DESIGN.md § Protocol).
func (s *Server) encodeRejected(rejected *roaring.Bitmap) []byte {
	arr := rejected.ToArray()
	handles := make([]Handle, len(arr))
	for i, v := range arr {
		handles[i] = Handle(v)
	}

	uuids := s.reg.ResolveHandles(handles)

	sorted := make([][16]byte, len(uuids))
	for i, u := range uuids {
		sorted[i] = [16]byte(u)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return uuidLess(sorted[i], sorted[j])
	})

	return deltauuid.EncodeSortedDeltas(sorted)
}

// uuidLess reports whether a sorts strictly before b, lexicographically by
// byte -- [16]byte has no built-in ordering, so this is a small helper
// rather than converting to a slice via bytes.Compare at every call site.
func uuidLess(a, b [16]byte) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// unixNanoToTime converts the proto request's "0 == unbounded" convention
// (§2) into TenantSet.Window's own zero-time-means-unbounded convention
// (tenant.go) -- the two are deliberately the same shape, so no further
// special-casing is needed beyond this one conversion.
func unixNanoToTime(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}
