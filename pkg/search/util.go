package search

import (
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

// jpe test?

func GetVirtualTagValues(tagName string) []string {
	switch tagName {

	case trace.StatusCodeTag:
		return []string{trace.StatusCodeUnset, trace.StatusCodeOK, trace.StatusCodeError}

	case trace.ErrorTag:
		return []string{"true"}
	}

	return nil
}

func GetVirtualTagValuesV2(tagName string) []tempopb.TagValue {

	switch tagName {
	case traceql.IntrinsicStatus.String():
		return []tempopb.TagValue{
			{Type: "keyword", Value: traceql.StatusOk.String()},
			{Type: "keyword", Value: traceql.StatusError.String()},
			{Type: "keyword", Value: traceql.StatusUnset.String()},
		}
	}

	return nil
}

// CombineSearchResults overlays the incoming search result with the existing result. This is required
// for the following reason:  a trace may be present in multiple blocks, or in partial segments
// in live traces.  The results should reflect elements of all segments.
func CombineSearchResults(existing *tempopb.TraceSearchMetadata, incoming *tempopb.TraceSearchMetadata) {
	if existing.TraceID == "" {
		existing.TraceID = incoming.TraceID
	}

	if existing.RootServiceName == "" {
		existing.RootServiceName = incoming.RootServiceName
	}

	if existing.RootTraceName == "" {
		existing.RootTraceName = incoming.RootTraceName
	}

	// Earliest start time.
	if existing.StartTimeUnixNano > incoming.StartTimeUnixNano {
		existing.StartTimeUnixNano = incoming.StartTimeUnixNano
	}

	// Longest duration
	if existing.DurationMs < incoming.DurationMs {
		existing.DurationMs = incoming.DurationMs
	}
}
