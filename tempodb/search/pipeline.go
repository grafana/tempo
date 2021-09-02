package search

import (
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
)

type tracefilter func(header *tempofb.SearchEntry) (matches bool)
type tagfilter func(c tempofb.TagContainer) (matches bool)

type Pipeline struct {
	tagfilters   []tagfilter
	tracefilters []tracefilter
}

func NewSearchPipeline(req *tempopb.SearchRequest) Pipeline {
	p := Pipeline{}

	if req.MinDurationMs > 0 {
		minDuration := uint64(time.Duration(req.MinDurationMs) * time.Millisecond)
		p.tracefilters = append(p.tracefilters, func(s *tempofb.SearchEntry) bool {
			return (s.EndTimeUnixNano() - s.StartTimeUnixNano()) >= minDuration
		})
	}

	if req.MaxDurationMs > 0 {
		maxDuration := uint64(time.Duration(req.MaxDurationMs) * time.Millisecond)
		p.tracefilters = append(p.tracefilters, func(s *tempofb.SearchEntry) bool {
			return (s.EndTimeUnixNano() - s.StartTimeUnixNano()) <= maxDuration
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

		p.tagfilters = append(p.tagfilters, func(s tempofb.TagContainer) bool {
			// Buffer is allocated here so function is thread-safe
			buffer := &tempofb.KeyValues{}

			// Must match all
			for i := range kb {
				if !tempofb.ContainsTag(s, buffer, kb[i], vb[i]) {
					return false
				}
			}
			return true
		})
	}

	return p
}

func (p *Pipeline) Matches(header *tempofb.SearchEntry) bool {

	for _, f := range p.tracefilters {
		if !f(header) {
			return false
		}
	}

	for _, f := range p.tagfilters {
		if !f(header) {
			return false
		}
	}

	return true
}

func (p *Pipeline) MatchesTags(c tempofb.TagContainer) bool {

	for _, f := range p.tagfilters {
		if !f(c) {
			return false
		}
	}

	return true
}
