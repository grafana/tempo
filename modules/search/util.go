package search

import (
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

func GetSearchResultFromData(s *tempofb.SearchData) *tempopb.TraceSearchMetadata {
	return &tempopb.TraceSearchMetadata{
		TraceID:           util.TraceIDToHexString(s.Id()),
		RootServiceName:   s.Get("root.service.name"),
		RootTraceName:     s.Get("root.name"),
		StartTimeUnixNano: s.StartTimeUnixNano(),
		DurationMs:        uint32((s.EndTimeUnixNano() - s.StartTimeUnixNano()) / 1_000_000),
	}
}
