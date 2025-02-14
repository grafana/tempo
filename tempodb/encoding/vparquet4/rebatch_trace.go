package vparquet4

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
	for i, rs := range trace.ResourceSpans {
		hash := resourceSpanHash(&rs)
		idx, ok := hashToIndex[hash]

		if !ok { // if the hash is unique, store the index
			hashToIndex[hash] = i
			uniqueIndexes = append(uniqueIndexes, i)

			continue
		}

		// else, merge the ScopeSpans with the existing identical ResourceSpan
		rebatchRS := &trace.ResourceSpans[idx]
		rebatchRS.ScopeSpans = append(rebatchRS.ScopeSpans, rs.ScopeSpans...)
	}

	// move unique ResourceSpans to the front and truncate the slice
	for i, idx := range uniqueIndexes {
		trace.ResourceSpans[i] = trace.ResourceSpans[idx]
	}
	trace.ResourceSpans = trace.ResourceSpans[:len(uniqueIndexes)]

	// rebatch ScopeSpans
	for i := range trace.ResourceSpans {
		rs := &trace.ResourceSpans[i]
		if len(rs.ScopeSpans) == 0 {
			continue
		}

		uniqueIndexes = uniqueIndexes[:0]
		clear(hashToIndex)

		for j, ss := range rs.ScopeSpans {
			hash := scopeSpanHash(&ss)
			idx, ok := hashToIndex[hash]

			if !ok { // if the hash is unique, store the index
				hashToIndex[hash] = j
				uniqueIndexes = append(uniqueIndexes, j)

				continue
			}

			// else, merge the Spans with the existing identical ScopeSpans
			rebatchSS := &rs.ScopeSpans[idx]
			rebatchSS.Spans = append(rebatchSS.Spans, ss.Spans...)
		}

		// move unique ScopeSpans to the front and truncate the slice
		for j, idx := range uniqueIndexes {
			rs.ScopeSpans[j] = rs.ScopeSpans[idx]
		}
		rs.ScopeSpans = rs.ScopeSpans[:len(uniqueIndexes)]
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
