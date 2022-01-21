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

const (
	notFound    = 1
	failedMatch = 2
	passedMatch = 3
)

func MatchesProto(id []byte, trace *tempopb.Trace, req *tempopb.SearchRequest) (*tempopb.TraceSearchMetadata, error) {
	traceStart := uint64(math.MaxUint64)
	traceEnd := uint64(0)

	traceMatch := notFound
	if len(req.Tags) == 0 {
		traceMatch = passedMatch
	}

	var rootSpan *v1.Span
	var rootBatch *v1.ResourceSpans
	// todo: is it possible to shortcircuit this loop?
	for _, b := range trace.Batches {
		// check if the batch matches. we will use the value of batchMatch later to determine if
		//  - we need to bother checking if the spans match
		//  - we can mark the entire trace as matching
		batchMatch := notFound
		if traceMatch != passedMatch {
			batchMatch = matchAttributes(req.Tags, b.Resource.Attributes)
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
				if traceMatch == passedMatch {
					continue
				}

				// if the batch explicitly doesn't match we don't have to bother
				if batchMatch == failedMatch {
					continue
				}

				// checks non attribute span properties for matching
				spanMatch := matchSpan(req.Tags, s)
				if spanMatch == failedMatch {
					continue
				}

				spanAttsMatch := matchAttributes(req.Tags, s.Attributes)
				// if batch, span attributes or the span itself match then the trace matches
				if spanAttsMatch == passedMatch || batchMatch == passedMatch || spanMatch == passedMatch {
					traceMatch = passedMatch
				}
			}
		}
	}

	if traceMatch != passedMatch {
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

// matchSpan searches for two reserved tag names "name" and "error" and maps them to actual
// properties of the span. it returns false if the span explicitly fails to match based on the
// passed conditions and true otherwise.
func matchSpan(tags map[string]string, s *v1.Span) int {
	match := notFound

	if name, ok := tags[search.SpanNameTag]; ok {
		if name != s.Name {
			return failedMatch
		} else {
			match = passedMatch
		}
	}

	if err, ok := tags[search.ErrorTag]; ok {
		if err == "true" && s.Status.Code != v1.Status_STATUS_CODE_ERROR {
			return failedMatch
		} else {
			match = passedMatch
		}
	}

	if status, ok := tags[search.StatusCodeTag]; ok {
		if search.StatusCodeMapping[status] != int(s.Status.Code) {
			return failedMatch
		} else {
			match = passedMatch
		}
	}

	return match
}

// matchAttributes returns an int that indicates if the passed tags match the trace attributes
//  possible values: attributesNotFound, attributesFailedMatch, attributesPassedMatch
func matchAttributes(tags map[string]string, atts []*v1common.KeyValue) int {
	// start with the assumption that we won't find any matching attributes
	allMatch := notFound

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

		// if an individual attribute failed to match we can bail here
		if !match {
			return failedMatch
		}

		// if we're here we've found a match but we need to continue searching in case
		//  a future tag fails to match
		allMatch = passedMatch
	}

	return allMatch
}
