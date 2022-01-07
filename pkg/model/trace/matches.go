package trace

import (
	"math"
	"strconv"
	"strings"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/search"
)

const RootSpanNotYetReceivedText = "<root span not yet received>"

func MatchesProto(id []byte, trace *tempopb.Trace, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	traceStart := uint64(math.MaxUint64)
	traceEnd := uint64(0)

	tagFound := false
	if len(req.Tags) == 0 {
		tagFound = true
	}

	var rootSpan *v1.Span
	var rootBatch *v1.ResourceSpans
	// todo: is it possible to shortcircuit this loop?
	for _, b := range trace.Batches {
		if !tagFound && searchAttributes(req.Tags, b.Resource.Attributes) {
			tagFound = true
		}

		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				if s.StartTimeUnixNano < traceStart {
					traceStart = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > traceEnd {
					traceEnd = s.EndTimeUnixNano
				}
				if rootSpan == nil && len(s.ParentSpanId) == 0 {
					rootSpan = s
					rootBatch = b
				}

				if tagFound {
					continue
				}

				if searchAttributes(req.Tags, s.Attributes) {
					tagFound = true
				}
			}
		}
	}

	if !tagFound {
		return nil, nil
	}

	traceStartMs := traceStart / 1000000
	traceEndMs := traceEnd / 1000000
	durationMs := uint32(traceEndMs - traceStartMs)
	if req.MaxDurationMs != 0 && req.MaxDurationMs < durationMs {
		return nil, nil
	}
	if req.MinDurationMs != 0 && req.MinDurationMs > durationMs {
		return nil, nil

	}
	if !(req.Start <= uint32(traceEndMs/1000) && req.End >= uint32(traceStartMs/1000)) {
		return nil, nil
	}

	// woohoo!
	rootServiceName := RootSpanNotYetReceivedText
	rootSpanName := RootSpanNotYetReceivedText
	if rootSpan != nil && rootBatch != nil {
		rootSpanName = rootSpan.Name

		for _, a := range rootBatch.Resource.Attributes {
			if a.Key == search.ServiceNameTag {
				rootServiceName = a.Value.GetStringValue()
				break
			}
		}
	}

	return &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(id),
		RootServiceName:   rootServiceName,
		RootTraceName:     rootSpanName,
		StartTimeUnixNano: traceStart,
		DurationMs:        durationMs,
	}, nil
}

// searchAttributes returns true if the tags passed are contained in the atts slice
func searchAttributes(tags map[string]string, atts []*v1common.KeyValue) bool {
	for _, a := range atts {
		var searchString string
		var ok bool

		if searchString, ok = tags[a.Key]; !ok {
			continue
		}

		// todo: support AnyValue_ArrayValue and AnyValue_KvlistValue
		switch v := a.Value.Value.(type) {
		case *v1common.AnyValue_StringValue:
			return strings.Contains(v.StringValue, searchString)
		case *v1common.AnyValue_IntValue:
			n, err := strconv.ParseInt(searchString, 10, 64)
			if err == nil {
				return v.IntValue == n
			}
		case *v1common.AnyValue_DoubleValue:
			f, err := strconv.ParseFloat(searchString, 64)
			if err == nil {
				return v.DoubleValue == f
			}
		case *v1common.AnyValue_BoolValue:
			b, err := strconv.ParseBool(searchString)
			if err == nil {
				return v.BoolValue == b
			}
		}
	}

	return false
}
