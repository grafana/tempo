// Package tracefingerprint computes and compares trace fingerprints, as
// defined by version 1 of the Tempo trace fingerprint contract.
//
// A span's identity is its resource service.name plus its span name. Along
// every path from a root down the trace, repeated identities in a row count
// as one, and each run of 1 to 3 consecutive identities is a shingle. The
// fingerprint is a 64-byte MinHash of the distinct shingles, and the shingle
// count is the number of distinct shingle hashes.
//
// The package does not depend on storage. Store [Version] with each
// fingerprint and pass it back when comparing.
package tracefingerprint

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"slices"
	"strings"

	"github.com/cespare/xxhash/v2"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

const (
	// Version is the fingerprint definition this package implements. Store it
	// with every fingerprint.
	Version = "v1"

	// Size is the length of a fingerprint in bytes.
	Size = slots / 8

	slots     = 512
	maxWindow = 3

	// seedLabel seeds coefficientA and coefficientB. The "v0" in it is what the
	// contract's reference vectors use.
	seedLabel = "trace-signature-v0/minhash-512x1"

	golden = 0x9e3779b97f4a7c15

	attrServiceName = "service.name"
	identityTag     = "NAME"
)

// coefficientA and coefficientB are the MinHash multipliers and offsets that
// every Fingerprinter shares read-only. They are part of definition v1:
// changing seedLabel or how coefficients derives them changes every
// fingerprint and needs a new Version.
var coefficientA, coefficientB = coefficients()

// Status says which fields of a Result are valid.
type Status uint8

const (
	// StatusUnavailable means there is no fingerprint and no shingle count,
	// for example because parent links form a cycle. It is the zero value, so
	// an unset Result is never valid.
	StatusUnavailable Status = iota
	// StatusEmpty means the trace has no spans: the shingle count is 0 and
	// there is no fingerprint.
	StatusEmpty
	// StatusComputed means Fingerprint and ShingleCount are both valid.
	StatusComputed
)

// Result is the fingerprint of one trace.
type Result struct {
	// Version is the definition that produced the result.
	Version string
	Status  Status
	// ShingleCount is the number of distinct shingle hashes. It is valid for
	// StatusEmpty, where it is 0, and for StatusComputed.
	ShingleCount uint32
	// Fingerprint is valid only for StatusComputed.
	Fingerprint [Size]byte
}

var (
	// ErrVersionMismatch means the two results have different versions.
	ErrVersionMismatch = errors.New("tracefingerprint: fingerprint versions differ")
	// ErrUnsupportedVersion means both results have a version this package
	// does not implement.
	ErrUnsupportedVersion = errors.New("tracefingerprint: unsupported fingerprint version")
	// ErrNoFingerprint means a result is nil or has no fingerprint, as with
	// StatusEmpty and StatusUnavailable.
	ErrNoFingerprint = errors.New("tracefingerprint: result has no fingerprint")
)

// Similarity estimates how similar two traces are, from 0 to 1, as the
// Jaccard similarity of their shingle sets. Both results need the same
// supported version and StatusComputed. Otherwise it returns an error,
// checked in this order: ErrNoFingerprint for a nil result,
// ErrVersionMismatch, ErrUnsupportedVersion, then ErrNoFingerprint for a
// result without a fingerprint. An error never means a score of 0.
func Similarity(a, b *Result) (float64, error) {
	if a == nil || b == nil {
		return 0, ErrNoFingerprint
	}
	if a.Version != b.Version {
		return 0, fmt.Errorf("%w: %q and %q", ErrVersionMismatch, a.Version, b.Version)
	}
	if a.Version != Version {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedVersion, a.Version)
	}
	if a.Status != StatusComputed || b.Status != StatusComputed {
		return 0, ErrNoFingerprint
	}

	differing := 0
	for i := 0; i < Size; i += 8 {
		differing += bits.OnesCount64(binary.LittleEndian.Uint64(a.Fingerprint[i:]) ^ binary.LittleEndian.Uint64(b.Fingerprint[i:]))
	}
	// Unrelated traces still match on about half the bits by chance, so
	// differing on half of them already scores 0.
	return max(0, 1-float64(differing)/(slots/2)), nil
}

// Fingerprinter computes fingerprints. It reuses its buffers between calls and
// does not allocate once they are large enough for the trace. The zero value
// is ready to use. Do not use one Fingerprinter from several goroutines, and
// do not copy it after first use: copies share its buffers.
type Fingerprinter struct {
	spans       []spanRef
	identities  []uint64
	firstChild  []int
	nextSibling []int
	stack       []frame
	shingles    []uint64
	buf         []byte
}

type spanRef struct {
	span    *v1.Span
	service string
}

// frame is one span on the current path. next is its next child to visit, or
// -1. window holds the spans of the last few distinct identities on the path,
// oldest first.
type frame struct {
	next   int
	n      int
	window [maxWindow]int
}

// Fingerprint computes the fingerprint of tr. It does not modify tr.
//
// Spans with the same span ID and kind count as one span. If their copies
// differ, the one with the smallest service, then span name, then parent ID
// is used, so span order and resource grouping do not change the result.
// Combined traces have one copy per span ID and kind, so this only applies
// to uncombined input.
//
// A span is a root if its parent span ID is empty or all zeros, matches no
// span, or is ambiguous. When two spans have the parent's span ID, as Zipkin
// client and server spans can, a server span's parent is the client one and
// any other span's parent is the server one. If that span is missing, or
// three or more spans have the parent's span ID, the parent is ambiguous.
//
// The result describes tr as given. To store a fingerprint that can be
// recomputed later, fingerprint the merged trace as storage will return it.
// The two can differ: vParquet5 reads a one-element string array stored in a
// dedicated service.name column back as a plain string.
//
// A nil or spanless trace gives StatusEmpty. A parent cycle, or more distinct
// shingles than fit in a uint32, gives StatusUnavailable.
func (f *Fingerprinter) Fingerprint(tr *tempopb.Trace) Result {
	res := Result{Version: Version}
	shingles, ok := f.shingleHashes(tr)
	switch {
	case !ok || uint64(len(shingles)) > math.MaxUint32:
		return res
	case len(shingles) == 0:
		res.Status = StatusEmpty
		return res
	}
	res.Status = StatusComputed
	res.ShingleCount = uint32(len(shingles))
	sketch(shingles, &res.Fingerprint)
	return res
}

// shingleHashes returns the sorted distinct shingle hashes of tr, or false if
// the trace has a parent cycle. The slice belongs to f and is valid until the
// next call.
func (f *Fingerprinter) shingleHashes(tr *tempopb.Trace) ([]uint64, bool) {
	f.shingles = f.shingles[:0]
	spans := f.collect(tr)
	defer clear(f.spans) // drop references to tr
	if len(spans) == 0 {
		return f.shingles, true
	}

	// Index len(spans) is a virtual parent of all roots. The stack is only the
	// current path, so it grows with trace depth, not width.
	root := len(spans)
	f.identities = slices.Grow(f.identities[:0], len(spans))[:len(spans)]
	f.firstChild = slices.Grow(f.firstChild[:0], len(spans)+1)[:len(spans)+1]
	f.nextSibling = slices.Grow(f.nextSibling[:0], len(spans))[:len(spans)]
	for i := range spans {
		f.identities[i] = f.identityHash(spans[i].service, spans[i].span.Name)
		f.firstChild[i] = -1
	}
	f.firstChild[root] = -1
	for i := range spans {
		parent := findParent(spans, spans[i].span)
		if parent < 0 {
			parent = root
		}
		f.nextSibling[i] = f.firstChild[parent]
		f.firstChild[parent] = i
	}

	f.stack = append(f.stack[:0], frame{next: f.firstChild[root]})
	visited := 0
	for len(f.stack) > 0 {
		top := &f.stack[len(f.stack)-1]
		c := top.next
		if c < 0 {
			f.stack = f.stack[:len(f.stack)-1]
			continue
		}
		top.next = f.nextSibling[c]
		visited++

		fr := frame{next: f.firstChild[c], n: top.n, window: top.window}
		if fr.n == 0 || !sameIdentity(spans, f.identities, fr.window[fr.n-1], c) {
			if fr.n == maxWindow {
				copy(fr.window[:], fr.window[1:])
				fr.n--
			}
			fr.window[fr.n] = c
			fr.n++
			for l := 1; l <= fr.n; l++ {
				f.shingles = append(f.shingles, f.windowHash(fr.window[fr.n-l:fr.n]))
			}
		}
		f.stack = append(f.stack, fr)
	}
	// Spans in a parent cycle can't be reached from a root, so any span not
	// visited means there is a cycle.
	if visited != len(spans) {
		return nil, false
	}

	slices.Sort(f.shingles)
	f.shingles = slices.Compact(f.shingles)
	return f.shingles, true
}

// collect returns the spans of tr sorted by span ID, then kind, with one span
// per span ID and kind (the key Tempo's trace combiner also uses). When copies
// differ it keeps the smallest by service, name and parent ID.
func (f *Fingerprinter) collect(tr *tempopb.Trace) []spanRef {
	f.spans = f.spans[:0]
	for _, rs := range tr.GetResourceSpans() {
		service := serviceName(rs.GetResource().GetAttributes())
		for _, ss := range rs.GetScopeSpans() {
			for _, span := range ss.GetSpans() {
				if span != nil {
					f.spans = append(f.spans, spanRef{span: span, service: service})
				}
			}
		}
	}

	slices.SortFunc(f.spans, func(a, b spanRef) int {
		if c := compareKey(a.span, b.span); c != 0 {
			return c
		}
		if c := strings.Compare(a.service, b.service); c != 0 {
			return c
		}
		if c := strings.Compare(a.span.Name, b.span.Name); c != 0 {
			return c
		}
		return bytes.Compare(a.span.ParentSpanId, b.span.ParentSpanId)
	})
	return slices.CompactFunc(f.spans, func(a, b spanRef) bool {
		return compareKey(a.span, b.span) == 0
	})
}

func compareKey(a, b *v1.Span) int {
	if c := bytes.Compare(a.SpanId, b.SpanId); c != 0 {
		return c
	}
	return cmp.Compare(a.Kind, b.Kind)
}

// findParent returns the index of span's parent, or -1 if the span is a root
// or its parent is missing or ambiguous, as Fingerprint documents. The
// client/server rule is the one in vParquet's findParentNodeInMap. When three
// or more spans share an ID, vParquet's nested set model gives up on the
// whole trace; here only the spans parented by that ID become roots, which
// matches how v1 treats a missing parent and keeps the rest of the trace.
func findParent(spans []spanRef, span *v1.Span) int {
	parentID := span.ParentSpanId
	if isRootID(parentID) {
		return -1
	}

	lo, found := slices.BinarySearchFunc(spans, parentID, func(s spanRef, id []byte) int {
		return bytes.Compare(s.span.SpanId, id)
	})
	if !found {
		return -1
	}
	// Three candidates already make the parent ambiguous, so stop counting.
	// Span kind can be any integer, so there could be many more.
	n := 1
	for n < 3 && lo+n < len(spans) && bytes.Equal(spans[lo+n].span.SpanId, parentID) {
		n++
	}

	switch n {
	case 1:
		return lo
	case 2:
		want := v1.Span_SPAN_KIND_SERVER
		if span.Kind == v1.Span_SPAN_KIND_SERVER {
			want = v1.Span_SPAN_KIND_CLIENT
		}
		if spans[lo].span.Kind == want {
			return lo
		}
		if spans[lo+1].span.Kind == want {
			return lo + 1
		}
	}
	return -1
}

func isRootID(id []byte) bool {
	for _, b := range id {
		if b != 0 {
			return false
		}
	}
	return true
}

// serviceName returns the last string service.name attribute, the value
// vParquet stores as the resource's service name. Non-string values are
// ignored.
func serviceName(attrs []*commonv1.KeyValue) string {
	service := ""
	for _, kv := range attrs {
		if kv.GetKey() != attrServiceName {
			continue
		}
		if s, ok := kv.GetValue().GetValue().(*commonv1.AnyValue_StringValue); ok && s != nil {
			service = s.StringValue
		}
	}
	return service
}

// identityHash hashes a span identity encoded as
// uvarint(len(service)) service uvarint(4) "NAME" uvarint(len(name)) name.
func (f *Fingerprinter) identityHash(service, name string) uint64 {
	buf := f.buf[:0]
	buf = binary.AppendUvarint(buf, uint64(len(service)))
	buf = append(buf, service...)
	buf = binary.AppendUvarint(buf, uint64(len(identityTag)))
	buf = append(buf, identityTag...)
	buf = binary.AppendUvarint(buf, uint64(len(name)))
	buf = append(buf, name...)
	f.buf = buf
	return xxhash.Sum64(buf)
}

func sameIdentity(spans []spanRef, identities []uint64, a, b int) bool {
	return identities[a] == identities[b] &&
		spans[a].service == spans[b].service &&
		spans[a].span.Name == spans[b].span.Name
}

// windowHash combines the identity hashes in window, ancestor first, into one
// shingle hash. Starting from the window length keeps windows of different
// lengths apart.
func (f *Fingerprinter) windowHash(window []int) uint64 {
	h := uint64(len(window)) * golden
	for _, i := range window {
		h = mix64(h ^ f.identities[i])
	}
	return h
}

// coefficients derives the MinHash multipliers and offsets from seedLabel
// with the SplitMix64 generator (https://prng.di.unimi.it/splitmix.c).
// Multipliers are made odd.
func coefficients() (a, b [slots]uint64) {
	state := xxhash.Sum64String(seedLabel)
	next := func() uint64 {
		state += golden
		return mix64(state)
	}
	for i := range a {
		a[i] = next() | 1
		b[i] = next()
	}
	return a, b
}

// sketch writes the one-bit MinHash of shingles into fp, which must be zero.
func sketch(shingles []uint64, fp *[Size]byte) {
	var mins [slots]uint64
	for i := range mins {
		mins[i] = math.MaxUint64
	}
	// Four shingles per pass over the slots load each slot's constants and
	// minimum once for all four, about 1.5 times as fast as one at a time
	// (see BenchmarkSketch).
	n := len(shingles) &^ 3
	for j := 0; j < n; j += 4 {
		x0, x1, x2, x3 := mix64(shingles[j]), mix64(shingles[j+1]), mix64(shingles[j+2]), mix64(shingles[j+3])
		for i := range mins {
			a, b := coefficientA[i], coefficientB[i]
			mins[i] = min(mins[i], x0*a+b, x1*a+b, x2*a+b, x3*a+b)
		}
	}
	for _, x := range shingles[n:] {
		x = mix64(x)
		for i := range mins {
			mins[i] = min(mins[i], x*coefficientA[i]+coefficientB[i])
		}
	}
	for i, m := range mins {
		fp[i/8] |= byte(mix64(m)&1) << (i % 8)
	}
}

// mix64 is the SplitMix64 finalizer from https://prng.di.unimi.it/splitmix.c.
func mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}
