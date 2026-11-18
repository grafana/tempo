package distributor

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Match pdata's protobuf recursion budget: KeyValue and AnyValue each consume
// one message, as do ArrayValue and KeyValueList. This is not a container count.
const maxAttributeMessageDepth = 100

var errAttributeDepth = errors.New("attribute value exceeded max recursion depth")

// Validate before protobuf sizing or conversion, which recursively visit values.
func validateTraceAttributeDepth(td ptrace.Traces) error {
	rss := td.ResourceSpans()
	for i := range rss.Len() {
		rs := rss.At(i)
		if !attributesWithinDepth(rs.Resource().Attributes(), 0) {
			return errAttributeDepth
		}
		sss := rs.ScopeSpans()
		for j := range sss.Len() {
			ss := sss.At(j)
			if !attributesWithinDepth(ss.Scope().Attributes(), 0) {
				return errAttributeDepth
			}
			spans := ss.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				if !attributesWithinDepth(span.Attributes(), 0) {
					return errAttributeDepth
				}
				events := span.Events()
				for n := 0; n < events.Len(); n++ {
					if !attributesWithinDepth(events.At(n).Attributes(), 0) {
						return errAttributeDepth
					}
				}
				links := span.Links()
				for n := 0; n < links.Len(); n++ {
					if !attributesWithinDepth(links.At(n).Attributes(), 0) {
						return errAttributeDepth
					}
				}
			}
		}
	}
	return nil
}

func attributesWithinDepth(attrs pcommon.Map, depth int) bool {
	if attrs.Len() == 0 {
		return true
	}
	if depth >= maxAttributeMessageDepth {
		return false
	}
	valid := true
	attrs.Range(func(_ string, value pcommon.Value) bool {
		valid = valueWithinDepth(value, depth+1)
		return valid
	})
	return valid
}

func valueWithinDepth(value pcommon.Value, depth int) bool {
	if depth >= maxAttributeMessageDepth {
		return false
	}
	switch value.Type() {
	case pcommon.ValueTypeSlice:
		if depth+1 >= maxAttributeMessageDepth {
			return false
		}
		values := value.Slice()
		for i := 0; i < values.Len(); i++ {
			if !valueWithinDepth(values.At(i), depth+2) {
				return false
			}
		}
	case pcommon.ValueTypeMap:
		if depth+1 >= maxAttributeMessageDepth {
			return false
		}
		return attributesWithinDepth(value.Map(), depth+2)
	}
	return true
}
