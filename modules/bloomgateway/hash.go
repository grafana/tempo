// Package bloomgateway implements the bloom gateway module described in
// DESIGN.md: a stateful, RF=1 service that partitions (trace, live block)
// pairs on a seeded hash of the trace ID and answers trace-by-id
// rejection-set queries with no false negatives.
package bloomgateway

import (
	"github.com/cespare/xxhash/v2"

	"github.com/grafana/tempo/pkg/util"
)

// Domain-separation tags for HashSeed / SeedFingerprint (AMENDMENT A4).
//
// The plan's original design derived both the effective hashing seed and
// the exposed seed_fingerprint as xxhash64(rawSeed) — meaning the
// "fingerprint" carried in every query response and snapshot header would
// literally BE the seed used to hash every trace ID, defeating the point of
// treating the seed as a secret (DESIGN.md § Design constraints: "the seed
// prevents tenants from crafting trace IDs that concentrate load...
// xxhash64 is trivially invertible without it").
//
// Fix: derive both values from the SAME raw seed but under different,
// fixed domain tags appended to the seed bytes before hashing:
//
//	hashSeed        = xxhash64(seed || 0x00)
//	SeedFingerprint = xxhash64(seed || 0x01)
//
// This scheme is frozen by the golden vectors in hash_test.go. It must
// never change without a coordinated reshard: every gateway instance (and,
// later, every query-frontend) must reproduce hashSeed byte-for-byte, and
// SeedFingerprint is compared byte-for-byte against snapshot headers and
// returned in every query response (DESIGN.md § Protocol, § Snapshots).
const (
	hashSeedDomainTag        byte = 0x00
	seedFingerprintDomainTag byte = 0x01
)

// HashSeed derives the effective xxhash64 seed used by Address from the raw
// configured secret (bloomgateway.Config.Seed). Callers derive this ONCE
// (typically at construction time, from flagext.Secret.String()) and reuse
// the resulting uint64 across every Address call — re-deriving it per call
// would double the hashing cost of the hot Add-apply and query paths for no
// benefit, since the derivation is a pure function of the (unchanging)
// configured seed.
//
// Domain-separated from SeedFingerprint; see the package-level doc comment
// above for the exact scheme.
func HashSeed(seed []byte) uint64 {
	return xxhash.Sum64(withDomainTag(seed, hashSeedDomainTag))
}

// SeedFingerprint is the drift-detection value carried in snapshot headers
// and every query response (DESIGN.md § Protocol: "lets the QF detect seed
// drift actively and alert, rather than inferring it from fallback rates";
// § Snapshots: "a seed fingerprint... to detect config mismatch"). It is
// derived from the raw seed under a domain tag distinct from HashSeed's, so
// disclosing it never discloses the value actually used to hash trace IDs.
func SeedFingerprint(seed []byte) uint64 {
	return xxhash.Sum64(withDomainTag(seed, seedFingerprintDomainTag))
}

// withDomainTag returns a freshly allocated copy of seed with tag appended.
// A fresh allocation (rather than append(seed, tag), which can silently
// mutate or alias the caller's backing array if it has spare capacity) is
// deliberate: seed is a secret and HashSeed/SeedFingerprint must never risk
// corrupting the caller's copy of it.
func withDomainTag(seed []byte, tag byte) []byte {
	tagged := make([]byte, len(seed)+1)
	copy(tagged, seed)
	tagged[len(seed)] = tag
	return tagged
}

// Address computes the leaf index and fingerprint for a trace ID under the
// given (already-derived, via HashSeed) hash seed, d, and f. traceID is
// canonicalized internally via util.PadTraceIDTo16Bytes before hashing —
// callers must NOT pre-pad and must NOT re-implement padding; conformance
// depends on there being exactly one padding call site
// (pkg/util.PadTraceIDTo16Bytes) upstream of every computed hash (DESIGN.md
// § Design constraints).
//
// Preconditions the caller (bloomgateway.Config.Validate) is responsible
// for upholding: d in [1, 32] and d+f <= 64. Address does not itself
// validate these — it is a pure, hot-path function — but degrades safely
// (never panics) if they are violated: out-of-range shifts saturate to 0
// per Go's unbounded-shift semantics for unsigned integers.
//
//   - leafIdx is the top d bits of the 64-bit hash: h >> (64 - d).
//   - fingerprint is the next f bits below the leaf index:
//     (h >> (64 - d - f)) & (1<<f - 1).
func Address(traceID []byte, hashSeed uint64, d, f uint8) (leafIdx uint32, fingerprint uint32) {
	canonical := util.PadTraceIDTo16Bytes(traceID)
	sum := sumWithSeed(canonical, hashSeed)

	leafIdx = uint32(sum >> (64 - uint64(d)))
	fingerprint = uint32((sum >> (64 - uint64(d) - uint64(f))) & fingerprintMask(f))
	return leafIdx, fingerprint
}

// fingerprintMask returns the low-f-bits mask 2^f - 1, saturating to
// ^uint64(0) for f >= 64 so callers can't trigger a shift-by->=64 (which
// Go's spec defines as yielding 0, turning "no mask" into "always zero"
// rather than panicking, but is still not the intended behavior).
func fingerprintMask(f uint8) uint64 {
	if f >= 64 {
		return ^uint64(0)
	}
	return (uint64(1) << f) - 1
}

// sumWithSeed hashes b under seed via a fresh *xxhash.Digest per call.
//
// A sync.Pool of *xxhash.Digest was tried and rejected: BenchmarkAddress
// (hash_bench_test.go) showed 0 allocs/op for a plain xxhash.NewWithSeed
// per call already — Digest is a small, pointer-free struct and escape
// analysis proves it never leaves this function, so it stack-allocates
// without a pool. Pooling only added sync.Pool's own Get/Put overhead on
// top: ~12 ns/op pooled vs. ~6.3 ns/op unpooled (Apple M4 Pro, go test
// -bench BenchmarkAddress -benchmem), i.e. pooling made the hot path ~2x
// slower for zero allocation benefit. Left documented here (rather than
// silently doing the "obviously right" thing) because §0 D4 explicitly
// asks this question and the intuitive answer is wrong for this type.
func sumWithSeed(b []byte, seed uint64) uint64 {
	d := xxhash.NewWithSeed(seed)
	_, _ = d.Write(b) // xxhash.Digest.Write never returns a non-nil error
	return d.Sum64()
}
