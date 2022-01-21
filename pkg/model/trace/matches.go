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

	// copy the tags from tempopb.SearchRequest. This map is then mutated by the two match* functions below
	// if, at the end of this loop, there are no more entries in the map then we matched all tags
	tagsToFind := map[string]string{}
	for k, v := range req.Tags {
		tagsToFind[k] = v
	}

	var rootSpan *v1.Span
	var rootBatch *v1.ResourceSpans
	// todo: is it possible to shortcircuit this loop?
	for _, b := range trace.Batches {
		if !allTagsFound(tagsToFind) {
			matchAttributes(tagsToFind, b.Resource.Attributes)
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

				// if the trace has already matched we don't have to bother
				if allTagsFound(tagsToFind) {
					continue
				}

				// checks non attribute span properties for matching
				matchSpan(tagsToFind, s)
				matchAttributes(tagsToFind, s.Attributes)
			}
		}
	}

	if !allTagsFound(tagsToFind) {
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

func allTagsFound(tagsToFind map[string]string) bool {
	return len(tagsToFind) == 0
}

// matchSpan searches for reserved tag names and maps them to actual
// properties of the span. it removes any matches it finds from
// the provided tags map
func matchSpan(tags map[string]string, s *v1.Span) {
	if name, ok := tags[search.SpanNameTag]; ok {
		if name == s.Name {
			delete(tags, search.SpanNameTag)
		}
	}

	if err, ok := tags[search.ErrorTag]; ok {
		if err == "true" && s.Status.Code == v1.Status_STATUS_CODE_ERROR {
			delete(tags, search.ErrorTag)
		}
	}

	if status, ok := tags[search.StatusCodeTag]; ok {
		if search.StatusCodeMapping[status] == int(s.Status.Code) {
			delete(tags, search.StatusCodeTag)
		}
	}
}

// matchAttributes tests to see if any tags in the map match any passed attributes
//  if it finds a match it removes the key from the map
func matchAttributes(tags map[string]string, atts []*v1common.KeyValue) {
	// start with the assumption that we won't find any matching attributes
	for _, a := range atts {
		var searchString string
		var ok bool

		if searchString, ok = tags[a.Key]; !ok {
			continue
		}

		match := false

		// todo: support AnyValue_ArrayValue and AnyValue_KvlistValue
		switch v := a.Value.Value.(type) {
		case *v1common.AnyValue_StringValue:
			match = strings.Contains(v.StringValue, searchString)
		case *v1common.AnyValue_IntValue:
			n, err := strconv.ParseInt(searchString, 10, 64)
			if err == nil {
				match = v.IntValue == n
			}
		case *v1common.AnyValue_DoubleValue:
			f, err := strconv.ParseFloat(searchString, 64)
			if err == nil {
				match = v.DoubleValue == f
			}
		case *v1common.AnyValue_BoolValue:
			b, err := strconv.ParseBool(searchString)
			if err == nil {
				match = v.BoolValue == b
			}
		}

		if match {
			delete(tags, a.Key)
		}
	}
}
