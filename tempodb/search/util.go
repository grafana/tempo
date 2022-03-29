package search

import (
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

func GetVirtualTags() []string {
	return []string{trace.ErrorTag}
}

func GetVirtualTagValues(tagName string) []string {
	switch tagName {

	case trace.StatusCodeTag:
		return []string{trace.StatusCodeUnset, trace.StatusCodeOK, trace.StatusCodeError}

	case trace.ErrorTag:
		return []string{"true"}
	}

	return nil
}

func GetSearchResultFromData(s *tempofb.SearchEntry) *tempopb.TraceSearchMetadata {
	return &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(s.Id()),
		RootServiceName:   s.Get(trace.RootServiceNameTag),
		RootTraceName:     s.Get(trace.RootSpanNameTag),
		StartTimeUnixNano: s.StartTimeUnixNano(),
		DurationMs:        uint32((s.EndTimeUnixNano() - s.StartTimeUnixNano()) / 1_000_000),
	}
}

// CombineResults overlays the incoming search result with the existing result. This is required
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
