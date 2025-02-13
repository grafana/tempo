package vparquet4

import (
	"math"
	"slices"
	"strings"

	"github.com/segmentio/fasthash/fnv1a"
)

// rebatchTrace removes redundant ResourceSpans and ScopeSpans from the trace through rebatching.
func rebatchTrace(trace *Trace) {
	if trace.ResourceSpans != nil {
		rsHashes := map[uint64]int{}
		uniqueRS := make([]ResourceSpans, 0, len(trace.ResourceSpans))

		for _, rs := range trace.ResourceSpans {
			hash := resourceSpanHash(&rs)
			resIdx, ok := rsHashes[hash]

			if !ok { // store the first of each resource span
				rsHashes[hash] = len(uniqueRS)
				uniqueRS = append(uniqueRS, rs)

				continue
			}

			uniqueRS[resIdx].ScopeSpans = append(uniqueRS[resIdx].ScopeSpans, rs.ScopeSpans...)
		}

		trace.ResourceSpans = uniqueRS
	}

	// now do the same for ScopeSpans
	ssHashes := map[uint64]int{}
	var uniqueSS []ScopeSpans

	for idx, rs := range trace.ResourceSpans {
		if rs.ScopeSpans != nil {
			clear(ssHashes)
			uniqueSS = make([]ScopeSpans, 0, len(rs.ScopeSpans))

			for _, ss := range rs.ScopeSpans {
				hash := scopeSpanHash(&ss)
				scopeIdx, ok := ssHashes[hash]

				if !ok { // store the first of each scope span
					ssHashes[hash] = len(uniqueSS)
					uniqueSS = append(uniqueSS, ss)

					continue
				}

				// combine into existing
				uniqueSS[scopeIdx].Spans = append(uniqueSS[scopeIdx].Spans, ss.Spans...)
			}

			trace.ResourceSpans[idx].ScopeSpans = uniqueSS
		}
	}
}

func scopeSpanHash(ss *ScopeSpans) uint64 {
	hash := fnv1a.HashString64(ss.Scope.Name)

	hash = fnv1a.AddString64(hash, ss.Scope.Version)

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

	hash = addHash(rs.Resource.Cluster, hash)
	hash = addHash(rs.Resource.Container, hash)
	hash = addHash(rs.Resource.K8sClusterName, hash)
	hash = addHash(rs.Resource.K8sContainerName, hash)
	hash = addHash(rs.Resource.K8sNamespaceName, hash)
	hash = addHash(rs.Resource.K8sPodName, hash)
	hash = addHash(rs.Resource.Namespace, hash)
	hash = addHash(rs.Resource.Pod, hash)

	hash = addHash(rs.Resource.DedicatedAttributes.String01, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String02, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String03, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String04, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String05, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String06, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String07, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String08, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String09, hash)
	hash = addHash(rs.Resource.DedicatedAttributes.String10, hash)

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

	// is array?
	for _, v := range attr.Value {
		hash = fnv1a.AddString64(hash, v)
	}
	for _, v := range attr.ValueBool {
		b := uint64(0)
		if v {
			b = 1
		}
		hash = fnv1a.AddUint64(hash, b)
	}
	for _, v := range attr.ValueDouble {
		hash = fnv1a.AddUint64(hash, math.Float64bits(v))
	}
	for _, v := range attr.ValueInt {
		hash = fnv1a.AddUint64(hash, uint64(v))
	}
	hash = addHash(attr.ValueUnsupported, hash)

	return hash
}

func addHash(s *string, hash uint64) uint64 {
	if s == nil {
		return hash
	}
	return fnv1a.AddString64(hash, *s)
}
