package search

import (
	"strings"
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

	if len(req.Tags) > 0 {
		// Convert all search params to bytes once
		kb := make([][]byte, 0, len(req.Tags))
		vb := make([][]byte, 0, len(req.Tags))

		for k, v := range req.Tags {
			kb = append(kb, []byte(strings.ToLower(k)))
			vb = append(vb, []byte(strings.ToLower(v)))
		}

		p.filters = append(p.filters, func(s *tempofb.SearchData) bool {
			// Buffer is allocated here so function is thread-safe
			buffer := &tempofb.KeyValues{}

			// Must match all
			for i := range kb {
				if !s.Contains(buffer, kb[i], vb[i]) {
					return false
				}
			}
			return true
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
