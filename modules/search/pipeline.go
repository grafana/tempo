package search

import (
	"time"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
)

type tracefilter func(header *tempofb.SearchData) (matches bool)

type Pipeline struct {
	filters []tracefilter
}

func NewSearchPipeline(req *tempopb.SearchRequest) Pipeline {
	p := Pipeline{}

	if len(req.Tags) > 0 {
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			// Must match all
			for k, v := range req.Tags {
				if !tempofb.SearchDataContains(s, k, v) {
					return false
				}
			}
			return true
		})
	}

	if req.MinDurationMs > 0 {
		minDuration := req.MinDurationMs * uint32(time.Millisecond)
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			return (s.EndTimeUnixNano()-s.StartTimeUnixNano())*uint64(time.Nanosecond) >= uint64(minDuration)
		})
	}

	if req.MaxDurationMs > 0 {
		maxDuration := req.MaxDurationMs * uint32(time.Millisecond)
		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			return (s.EndTimeUnixNano()-s.StartTimeUnixNano())*uint64(time.Nanosecond) <= uint64(maxDuration)
		})
	}

	return p
}

func (p *Pipeline) Matches(header *tempofb.SearchData) bool {

	for _, f := range p.filters {
		if !f(header) {
			return false
		}
	}

	return true
}

func (p *Pipeline) MatchesAny(headers []*tempofb.SearchData) bool {

	if len(p.filters) == 0 {
		// Empty pipeline matches everything
		return true
	}

	for _, h := range headers {
		for _, f := range p.filters {
			if f(h) {
				return true
			}
		}
	}

	return false
}
