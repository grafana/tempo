// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"encoding/base64"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// builderPool reduces allocations in the hot path by reusing string builders.
var builderPool = sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// buildGroupKey assembles the grouping key for a span using its name,
// status, and configured attribute matches. A pooled builder minimizes
// allocations in this frequently executed path.
func (p *spanPruningProcessor) buildGroupKey(span ptrace.Span) string {
	builder := builderPool.Get().(*strings.Builder)
	builder.Reset()
	defer builderPool.Put(builder)

	builder.WriteString(span.Name())

	// Include span kind in grouping key
	builder.WriteString("|kind=")
	builder.WriteString(span.Kind().String())

	// Include status code in grouping key
	builder.WriteString("|status=")
	builder.WriteString(span.Status().Code().String())

	// Include TraceState for Consistent Probability Sampling (CPS) compatibility.
	// Spans with different TraceState values (e.g., different sampling thresholds)
	// represent different sampling populations and must not be aggregated together.
	builder.WriteString("|ts=")
	builder.WriteString(span.TraceState().AsRaw())

	attrs := span.Attributes()

	// Collect all matching attribute key-value pairs
	matchedAttrs := make(map[string]pcommon.Value)
	attrs.Range(func(key string, value pcommon.Value) bool {
		for _, pattern := range p.attributePatterns {
			if pattern.glob.Match(key) {
				matchedAttrs[key] = value
				break // Only match each key once
			}
		}
		return true
	})

	// Sort keys for consistent ordering in the group key
	keys := make([]string, 0, len(matchedAttrs))
	for k := range matchedAttrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the group key with sorted attribute key-value pairs
	for _, key := range keys {
		builder.WriteString("|")
		builder.WriteString(key)
		builder.WriteString("=")
		writeAttributeValueKey(builder, matchedAttrs[key])
	}

	return builder.String()
}

func writeAttributeValueKey(builder *strings.Builder, value pcommon.Value) {
	builder.WriteString(value.Type().String())
	builder.WriteString(":")
	switch value.Type() {
	case pcommon.ValueTypeEmpty:
		return
	case pcommon.ValueTypeStr:
		builder.WriteString(value.Str())
	case pcommon.ValueTypeBool:
		builder.WriteString(strconv.FormatBool(value.Bool()))
	case pcommon.ValueTypeInt:
		builder.WriteString(strconv.FormatInt(value.Int(), 10))
	case pcommon.ValueTypeDouble:
		builder.WriteString(strconv.FormatFloat(value.Double(), 'g', -1, 64))
	case pcommon.ValueTypeBytes:
		bytesValue := value.Bytes().AsRaw()
		builder.WriteString(base64.StdEncoding.EncodeToString(bytesValue))
	case pcommon.ValueTypeMap:
		writeAttributeMapKey(builder, value.Map())
	case pcommon.ValueTypeSlice:
		writeAttributeSliceKey(builder, value.Slice())
	}
}

func writeAttributeMapKey(builder *strings.Builder, value pcommon.Map) {
	keys := make([]string, 0, value.Len())
	value.Range(func(key string, _ pcommon.Value) bool {
		keys = append(keys, key)
		return true
	})
	sort.Strings(keys)
	builder.WriteString("{")
	for index, key := range keys {
		if index > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		attrValue, _ := value.Get(key)
		writeAttributeValueKey(builder, attrValue)
	}
	builder.WriteString("}")
}

func writeAttributeSliceKey(builder *strings.Builder, value pcommon.Slice) {
	builder.WriteString("[")
	for index := 0; index < value.Len(); index++ {
		if index > 0 {
			builder.WriteString(",")
		}
		writeAttributeValueKey(builder, value.At(index))
	}
	builder.WriteString("]")
}

// buildParentGroupKey constructs a parent grouping key from name and status
// only; attributes are intentionally excluded for parent aggregation. Depth is
// required to avoid duplicate names and status entries overwriting each other.
func (*spanPruningProcessor) buildParentGroupKey(span ptrace.Span, depth int) string {
	builder := builderPool.Get().(*strings.Builder)
	builder.Reset()
	defer builderPool.Put(builder)

	builder.WriteString(strconv.Itoa(depth))
	builder.WriteByte('|')
	builder.WriteString(span.Name())
	builder.WriteString("|kind=")
	builder.WriteString(span.Kind().String())
	builder.WriteString("|status=")
	builder.WriteString(span.Status().Code().String())
	// Include TraceState for CPS compatibility
	builder.WriteString("|ts=")
	builder.WriteString(span.TraceState().AsRaw())
	return builder.String()
}

// buildLeafGroupKey derives a leaf grouping key that includes the parent's
// span name (if present) plus the standard grouping key, caching results per
// node to avoid recomputation.
func (p *spanPruningProcessor) buildLeafGroupKey(node *spanNode) string {
	// Use cached group key if available
	if node.groupKey != "" {
		return node.groupKey
	}

	builder := builderPool.Get().(*strings.Builder)
	builder.Reset()
	defer builderPool.Put(builder)

	// Include parent span name to separate groups by parent
	if node.parent != nil {
		builder.WriteString("parent=")
		builder.WriteString(node.parent.span.Name())
		builder.WriteString("|")
	}

	// Include regular group key (name + status + attributes)
	builder.WriteString(p.buildGroupKey(node.span))

	// Cache the key for future use
	node.groupKey = builder.String()
	return node.groupKey
}

// groupLeafNodesByKey groups leaf nodes by their derived key so that spans
// with identical grouping characteristics can be aggregated together.
func (p *spanPruningProcessor) groupLeafNodesByKey(leafNodes []*spanNode) map[string][]*spanNode {
	// Pre-size map based on expected number of groups (assume ~1/4 unique groups)
	groups := make(map[string][]*spanNode, len(leafNodes)/4+1)
	for _, node := range leafNodes {
		key := p.buildLeafGroupKey(node)
		groups[key] = append(groups[key], node)
	}
	return groups
}
