package trace

import (
	"strconv"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	common_v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
)

type ExtractTagFunc func(tag string) bool

// ExtractSearchData returns the flatbuffer search data for the given trace.  It is extracted here
// in the distributor because this is the only place on the ingest path where the trace is available
// in object form.
func ExtractSearchData(tr *tempopb.Trace, id []byte, extractTag ExtractTagFunc) []byte {
	data := &tempofb.SearchEntryMutable{}

	data.TraceID = id

	for _, b := range tr.Batches {
		// Batch attrs
		if b.Resource != nil {
			for _, a := range b.Resource.Attributes {
				if !extractTag(a.Key) {
					continue
				}
				if s, ok := extractValueAsString(a.Value); ok {
					data.AddTag(a.Key, s)
				}
			}
		}

		for _, ils := range b.ScopeSpans {
			for _, s := range ils.Spans {

				// Root span
				if len(s.ParentSpanId) == 0 {

					// Collect root.name
					data.AddTag(RootSpanNameTag, s.Name)

					// Collect root.service.name
					if b.Resource != nil {
						for _, a := range b.Resource.Attributes {
							if a.Key == ServiceNameTag {
								if s, ok := extractValueAsString(a.Value); ok {
									data.AddTag(RootServiceNameTag, s)
								}
							}
						}
					}
				}

				// Collect for any spans
				data.AddTag(SpanNameTag, s.Name)
				if s.Status != nil {
					data.AddTag(StatusCodeTag, strconv.Itoa(int(s.Status.Code)))
				}
				data.SetStartTimeUnixNano(s.StartTimeUnixNano)
				data.SetEndTimeUnixNano(s.EndTimeUnixNano)

				for _, a := range s.Attributes {
					if !extractTag(a.Key) {
						continue
					}
					if s, ok := extractValueAsString(a.Value); ok {
						data.AddTag(a.Key, s)
					}
				}
			}
		}
	}

	return data.ToBytes()
}

func extractValueAsString(v *common_v1.AnyValue) (s string, ok bool) {
	vv := v.GetValue()
	if vv == nil {
		return "", false
	}

	if s, ok := vv.(*common_v1.AnyValue_StringValue); ok {
		return s.StringValue, true
	}

	if b, ok := vv.(*common_v1.AnyValue_BoolValue); ok {
		return strconv.FormatBool(b.BoolValue), true
	}

	if i, ok := vv.(*common_v1.AnyValue_IntValue); ok {
		return strconv.FormatInt(i.IntValue, 10), true
	}

	if d, ok := vv.(*common_v1.AnyValue_DoubleValue); ok {
		return strconv.FormatFloat(d.DoubleValue, 'g', -1, 64), true
	}

	return "", false
}
