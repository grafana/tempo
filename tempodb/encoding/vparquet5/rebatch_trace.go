package vparquet5

import (
	"math"
	"slices"
	"strings"

	"github.com/segmentio/fasthash/fnv1a"
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
	hash := fnv1a.HashString64(ss.Scope.Name)
	hash = addHashSeparator(hash)
	hash = fnv1a.AddString64(hash, ss.Scope.Version)
	hash = addHashSeparator(hash)

	// sort keys for consistency
	slices.SortFunc(ss.Scope.Attrs, func(i, j Attribute) int {
		return strings.Compare(i.Key, j.Key)
	})

	for _, attr := range ss.Scope.Attrs {
		hash = attributeHash(&attr, hash)
	}

	return hash
}

func resourceSpanHash(rs *ResourceSpans) uint64 {
	hash := fnv1a.HashString64(rs.Resource.ServiceName)
	hash = addHashSeparator(hash)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String01...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String02...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String03...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String04...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String05...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String06...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String07...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String08...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String09...)
	hash = addHashStr(hash, rs.Resource.DedicatedAttributes.String10...)

	hash = addHashInt(hash, rs.Resource.DedicatedAttributes.Int01...)
	hash = addHashInt(hash, rs.Resource.DedicatedAttributes.Int02...)
	hash = addHashInt(hash, rs.Resource.DedicatedAttributes.Int03...)
	hash = addHashInt(hash, rs.Resource.DedicatedAttributes.Int04...)
	hash = addHashInt(hash, rs.Resource.DedicatedAttributes.Int05...)

	// sort keys for consistency
	slices.SortFunc(rs.Resource.Attrs, func(i, j Attribute) int {
		return strings.Compare(i.Key, j.Key)
	})

	for _, attr := range rs.Resource.Attrs {
		hash = attributeHash(&attr, hash)
	}

	return hash
}

func attributeHash(attr *Attribute, hash uint64) uint64 {
	hash = fnv1a.AddString64(hash, attr.Key)

	if attr.IsArray {
		hash = addHashSeparator(hash)
	}
	hash = addHashStr(hash, attr.Value...)
	hash = addHashInt(hash, attr.ValueInt...)
	hash = addHashDouble(hash, attr.ValueDouble...)
	hash = addHashBool(hash, attr.ValueBool...)

	if attr.ValueUnsupported != nil {
		hash = addHashStr(hash, *attr.ValueUnsupported)
	} else {
		hash = addHashSeparator(hash)
	}

	return hash
}

func addHashStr(hash uint64, strs ...string) uint64 {
	if len(strs) == 0 {
		return addHashSeparator(hash)
	}
	for _, s := range strs {
		hash = addHashSeparator(hash)
		hash = fnv1a.AddString64(hash, s)
	}
	return hash
}

func addHashInt(hash uint64, ints ...int64) uint64 {
	if len(ints) == 0 {
		return addHashSeparator(hash)
	}
	for _, n := range ints {
		hash = fnv1a.AddUint64(hash, uint64(n))
	}
	return hash
}

func addHashDouble(hash uint64, ints ...float64) uint64 {
	if len(ints) == 0 {
		return addHashSeparator(hash)
	}
	for _, n := range ints {
		hash = fnv1a.AddUint64(hash, math.Float64bits(n))
	}
	return hash
}

func addHashBool(hash uint64, bools ...bool) uint64 {
	if len(bools) == 0 {
		return addHashSeparator(hash)
	}
	for _, b := range bools {
		if b {
			hash = fnv1a.AddUint64(hash, 1)
		} else {
			hash = fnv1a.AddUint64(hash, 0)
		}
	}
	return hash
}

func addHashSeparator(hash uint64) uint64 {
	// hash twice with large primes to avoid collisions
	hash = fnv1a.AddUint64(hash, 9952039)
	return fnv1a.AddUint64(hash, 10188397)
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
