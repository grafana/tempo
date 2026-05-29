package vparquet5

import (
	"math"
	"slices"
	"strings"

	"github.com/grafana/tempo/pkg/hash"
)

// rebatchTrace removes redundant ResourceSpans and ScopeSpans from the trace through rebatching.
func rebatchTrace(trace *Trace) {
	if len(trace.ResourceSpans) == 0 {
		return
	}

	// preallocate a map and a slice to collect the indices of unique ResourceSpans and ScopeSpans
	uniqueIndexes := make([]int, 0, max(len(trace.ResourceSpans), len(trace.ResourceSpans[0].ScopeSpans)))
	hashToIndex := make(map[uint64]int, len(uniqueIndexes))

	// rebatch ResourceSpans
	for i := range trace.ResourceSpans {
		rs := &trace.ResourceSpans[i]

		hash := resourceSpanHash(rs)
		idx, ok := hashToIndex[hash]

		if !ok { // if the hash is unique, store the index
			hashToIndex[hash] = i
			uniqueIndexes = append(uniqueIndexes, i)

			continue
		}

		// else, merge the ScopeSpans with the existing identical ResourceSpan
		rebatchRS := &trace.ResourceSpans[idx]
		rebatchRS.ScopeSpans = append(rebatchRS.ScopeSpans, rs.ScopeSpans...)

		// the append above created copies of ScopeSpans, we have to clear the originals
		// otherwise we will have multiple copies of the same slices in different ScopeSpans
		rs.ScopeSpans = clearScopeSpans(rs.ScopeSpans)
	}

	// move unique ResourceSpans to the front and truncate the slice
	if len(uniqueIndexes) < len(trace.ResourceSpans) {
		for i, idx := range uniqueIndexes {
			if i != idx {
				trace.ResourceSpans[i], trace.ResourceSpans[idx] = trace.ResourceSpans[idx], trace.ResourceSpans[i]
			}
		}
		trace.ResourceSpans = trace.ResourceSpans[:len(uniqueIndexes)]
	}

	// rebatch ScopeSpans
	for i := range trace.ResourceSpans {
		rs := &trace.ResourceSpans[i]
		if len(rs.ScopeSpans) < 2 {
			continue
		}

		uniqueIndexes = uniqueIndexes[:0]
		clear(hashToIndex)

		for j := range rs.ScopeSpans {
			ss := &rs.ScopeSpans[j]

			hash := scopeSpanHash(ss)
			idx, ok := hashToIndex[hash]

			if !ok { // if the hash is unique, store the index
				hashToIndex[hash] = j
				uniqueIndexes = append(uniqueIndexes, j)

				continue
			}

			// else, merge the Spans with the existing identical ScopeSpans
			rebatchSS := &rs.ScopeSpans[idx]
			rebatchSS.Spans = append(rebatchSS.Spans, ss.Spans...)
			rebatchSS.SpanCount = int32(len(rebatchSS.Spans))

			// the append above creates copies of Spans, we have to clear the originals otherwise
			// we will have multiple copies of the same slices in different Spans
			ss.Spans = clearSpans(ss.Spans)
		}

		// move unique ScopeSpans to the front and truncate the slice
		if len(uniqueIndexes) < len(rs.ScopeSpans) {
			for j, idx := range uniqueIndexes {
				if j != idx {
					rs.ScopeSpans[j], rs.ScopeSpans[idx] = rs.ScopeSpans[idx], rs.ScopeSpans[j]
				}
			}
			rs.ScopeSpans = rs.ScopeSpans[:len(uniqueIndexes)]
		}
	}

	// Sort spans by time
	/*for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			slices.SortFunc(ss.Spans, func(i, j Span) int {
				return cmp.Compare(i.StartTimeUnixNano, j.StartTimeUnixNano)
			})
		}
	}*/
}

func scopeSpanHash(ss *ScopeSpans) uint64 {
	d := hash.New()
	_, _ = d.WriteString(ss.Scope.Name)
	addHashSeparator(d)
	_, _ = d.WriteString(ss.Scope.Version)
	addHashSeparator(d)

	// sort keys for consistency
	slices.SortFunc(ss.Scope.Attrs, func(i, j Attribute) int {
		return strings.Compare(i.Key, j.Key)
	})

	for _, attr := range ss.Scope.Attrs {
		attributeHash(d, &attr)
	}

	return d.Sum64()
}

func resourceSpanHash(rs *ResourceSpans) uint64 {
	d := hash.New()
	_, _ = d.WriteString(rs.Resource.ServiceName)
	addHashSeparator(d)
	addHashStr(d, rs.Resource.DedicatedAttributes.String01...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String02...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String03...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String04...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String05...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String06...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String07...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String08...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String09...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String10...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String11...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String12...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String13...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String14...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String15...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String16...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String17...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String18...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String19...)
	addHashStr(d, rs.Resource.DedicatedAttributes.String20...)

	addHashInt(d, rs.Resource.DedicatedAttributes.Int01...)
	addHashInt(d, rs.Resource.DedicatedAttributes.Int02...)
	addHashInt(d, rs.Resource.DedicatedAttributes.Int03...)
	addHashInt(d, rs.Resource.DedicatedAttributes.Int04...)
	addHashInt(d, rs.Resource.DedicatedAttributes.Int05...)

	// sort keys for consistency
	slices.SortFunc(rs.Resource.Attrs, func(i, j Attribute) int {
		return strings.Compare(i.Key, j.Key)
	})

	for _, attr := range rs.Resource.Attrs {
		attributeHash(d, &attr)
	}

	return d.Sum64()
}

func attributeHash(d *hash.Digest, attr *Attribute) {
	_, _ = d.WriteString(attr.Key)

	if attr.IsArray {
		addHashSeparator(d)
	}
	addHashStr(d, attr.Value...)
	addHashInt(d, attr.ValueInt...)
	addHashDouble(d, attr.ValueDouble...)
	addHashBool(d, attr.ValueBool...)

	if attr.ValueUnsupported != nil {
		addHashStr(d, *attr.ValueUnsupported)
	} else {
		addHashSeparator(d)
	}
}

func addHashStr(d *hash.Digest, strs ...string) {
	if len(strs) == 0 {
		addHashSeparator(d)
		return
	}
	for _, s := range strs {
		addHashSeparator(d)
		_, _ = d.WriteString(s)
	}
}

func addHashInt(d *hash.Digest, ints ...int64) {
	if len(ints) == 0 {
		addHashSeparator(d)
		return
	}
	for _, n := range ints {
		d.WriteUint64(uint64(n))
	}
}

func addHashDouble(d *hash.Digest, ints ...float64) {
	if len(ints) == 0 {
		addHashSeparator(d)
		return
	}
	for _, n := range ints {
		d.WriteUint64(math.Float64bits(n))
	}
}

func addHashBool(d *hash.Digest, bools ...bool) {
	if len(bools) == 0 {
		addHashSeparator(d)
		return
	}
	for _, b := range bools {
		if b {
			d.WriteUint64(1)
		} else {
			d.WriteUint64(0)
		}
	}
}

func addHashSeparator(d *hash.Digest) {
	// hash twice with large primes to avoid collisions
	d.WriteUint64(9952039)
	d.WriteUint64(10188397)
}

// clearScopeSpans clears slices in ScopeSpans so avoid multiple copies of the same
// slice in different ScopeSpans.
func clearScopeSpans(sss []ScopeSpans) []ScopeSpans {
	sss = sss[:cap(sss)]
	for i := range sss {
		ss := &sss[i]
		ss.Scope.Attrs = nil
		ss.Spans = nil
		ss.SpanCount = 0
	}
	return sss[:0]
}

// clearSpans clears slices in spans to avoid multiple copies of the same
// slice in different Spans.
func clearSpans(spans []Span) []Span {
	spans = spans[:cap(spans)]
	for i := range spans {
		s := &spans[i]
		s.Attrs = nil
		s.Events = nil
		s.Links = nil
	}
	return spans[:0]
}
