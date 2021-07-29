package search

import (
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

const (
	RootServiceNameTag = "root.service.name"
	ServiceNameTag     = "service.name"
	RootSpanPrefix     = "root."
	RootSpanNameTag    = "root.name"
	SpanNameTag        = "name"
)

func GetSearchResultFromData(s *tempofb.SearchData) *tempopb.TraceSearchMetadata {
	return &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(s.Id()),
		RootServiceName:   s.Get(RootServiceNameTag),
		RootTraceName:     s.Get(RootSpanNameTag),
		StartTimeUnixNano: s.StartTimeUnixNano(),
		DurationMs:        uint32((s.EndTimeUnixNano() - s.StartTimeUnixNano()) / 1_000_000),
	}
}
