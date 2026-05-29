package vparquet4

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
}

func scopeSpanHash(ss *ScopeSpans) uint64 {
	d := hash.New()
	_, _ = d.WriteString(ss.Scope.Name)
	_, _ = d.WriteString(ss.Scope.Version)

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

	addHash(d, rs.Resource.Cluster)
	addHash(d, rs.Resource.Container)
	addHash(d, rs.Resource.K8sClusterName)
	addHash(d, rs.Resource.K8sContainerName)
	addHash(d, rs.Resource.K8sNamespaceName)
	addHash(d, rs.Resource.K8sPodName)
	addHash(d, rs.Resource.Namespace)
	addHash(d, rs.Resource.Pod)

	addHash(d, rs.Resource.DedicatedAttributes.String01)
	addHash(d, rs.Resource.DedicatedAttributes.String02)
	addHash(d, rs.Resource.DedicatedAttributes.String03)
	addHash(d, rs.Resource.DedicatedAttributes.String04)
	addHash(d, rs.Resource.DedicatedAttributes.String05)
	addHash(d, rs.Resource.DedicatedAttributes.String06)
	addHash(d, rs.Resource.DedicatedAttributes.String07)
	addHash(d, rs.Resource.DedicatedAttributes.String08)
	addHash(d, rs.Resource.DedicatedAttributes.String09)
	addHash(d, rs.Resource.DedicatedAttributes.String10)

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

	// is array?
	for _, v := range attr.Value {
		_, _ = d.WriteString(v)
	}
	for _, v := range attr.ValueBool {
		b := uint64(0)
		if v {
			b = 1
		}
		d.WriteUint64(b)
	}
	for _, v := range attr.ValueDouble {
		d.WriteUint64(math.Float64bits(v))
	}
	for _, v := range attr.ValueInt {
		d.WriteUint64(uint64(v))
	}
	addHash(d, attr.ValueUnsupported)
}

func addHash(d *hash.Digest, s *string) {
	if s == nil {
		// hash twice with large primes to avoid collisions
		d.WriteUint64(9952039)
		d.WriteUint64(10188397)
		return
	}
	_, _ = d.WriteString(*s)
}

// clearScopeSpans clears slices in ScopeSpans so avoid multiple copies of the same
// slice in different ScopeSpans.
func clearScopeSpans(sss []ScopeSpans) []ScopeSpans {
	sss = sss[:cap(sss)]
	for i := range sss {
		ss := &sss[i]
		ss.Scope.Attrs = nil
		ss.Spans = nil
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
