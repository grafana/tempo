package search

import (
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

const (
	RootServiceNameTag = "root.service.name"
	ServiceNameTag     = "service.name"
	RootSpanNameTag    = "root.name"
	SpanNameTag        = "name"
	ErrorTag           = "error"
	StatusCodeTag      = "status.code"
	StatusCodeUnset    = "unset"
	StatusCodeOK       = "ok"
	StatusCodeError    = "error"
)

var StatusCodeMapping = map[string]int{
	StatusCodeUnset: int(v1.Status_STATUS_CODE_UNSET),
	StatusCodeOK:    int(v1.Status_STATUS_CODE_OK),
	StatusCodeError: int(v1.Status_STATUS_CODE_ERROR),
}

func GetVirtualTags() []string {
	return []string{ErrorTag}
}

func GetVirtualTagValues(tagName string) []string {
	switch tagName {

	case StatusCodeTag:
		return []string{StatusCodeUnset, StatusCodeOK, StatusCodeError}

	case ErrorTag:
		return []string{"true"}
	}

	return nil
}

func GetSearchResultFromData(s *tempofb.SearchEntry) *tempopb.TraceSearchMetadata {
	return &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(s.Id()),
		RootServiceName:   s.Get(RootServiceNameTag),
		RootTraceName:     s.Get(RootSpanNameTag),
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
