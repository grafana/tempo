package search

import (
	"strconv"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

const SecretExhaustiveSearchTag = "x-dbg-exhaustive"

type tracefilter func(entry tempofb.Trace) (matches bool)
type tagfilter func(page tempofb.TagContainer) (matches bool)
type blockfilter func(header tempofb.Block) (matches bool)

type Pipeline struct {
	blockfilters []blockfilter
	tagfilters   []tagfilter // shared by pages and traces
	tracefilters []tracefilter
}

func NewSearchPipeline(req *tempopb.SearchRequest) Pipeline {
	p := Pipeline{}

	if req.MinDurationMs > 0 {
		minDurationNanos := uint64(time.Duration(req.MinDurationMs) * time.Millisecond)

		p.tracefilters = append(p.tracefilters, func(s tempofb.Trace) bool {
			et := s.EndTimeUnixNano()
			st := s.StartTimeUnixNano()
			return (et - st) >= minDurationNanos
		})

		p.blockfilters = append(p.blockfilters, func(s tempofb.Block) bool {
			max := s.MaxDurationNanos()
			return max >= minDurationNanos
		})
	}

	if req.MaxDurationMs > 0 {
		maxDurationNanos := uint64(time.Duration(req.MaxDurationMs) * time.Millisecond)

		p.tracefilters = append(p.tracefilters, func(s tempofb.Trace) bool {
			et := s.EndTimeUnixNano()
			st := s.StartTimeUnixNano()
			return (et - st) <= maxDurationNanos
		})

		p.blockfilters = append(p.blockfilters, func(s tempofb.Block) bool {
			min := s.MinDurationNanos()
			return min <= maxDurationNanos
		})
	}

	if req.Start != 0 && req.End != 0 {
		p.tracefilters = append(p.tracefilters, func(s tempofb.Trace) bool {
			// req.Start and req.End are in unix epoch seconds
			startTimeSeconds := uint32(s.StartTimeUnixNano() / uint64(time.Second))
			endTimeSeconds := uint32(s.EndTimeUnixNano() / uint64(time.Second))

			return req.Start <= endTimeSeconds && req.End >= startTimeSeconds
		})

		// todo: add block level filter for start/end time
	}

	if len(req.Tags) > 0 {
		// Convert all search params to bytes once
		kb := make([][]byte, 0, len(req.Tags))
		vb := make([][]byte, 0, len(req.Tags))

		for k, v := range req.Tags {
			skip, k, v := p.rewriteTagLookup(k, v)
			if skip {
				continue
			}

			kb = append(kb, []byte(strings.ToLower(k)))
			vb = append(vb, []byte(strings.ToLower(v)))
		}

		p.tagfilters = append(p.tagfilters, func(s tempofb.TagContainer) bool {
			// Buffer is allocated here so pipeline can be used concurrently.
			buffer := &tempofb.KeyValues{}

			// Must match all
			for i := range kb {
				if !s.Contains(kb[i], vb[i], buffer) {
					return false
				}
			}
			return true
		})
	}

	return p
}

// rewriteTagLookup intercepts certain tag/value lookups and rewrites them. It returns
// true if the tag lookup should be excluded from the remaining tag/value lookups because
// the it was rewritten into a different filter altogether.  Otherwise it returns false,
// and a new set of tag/value strings to use, which will either be the original inputs
// or rewritten lookups.
func (p *Pipeline) rewriteTagLookup(k, v string) (skip bool, newk, newv string) {
	switch k {
	case SecretExhaustiveSearchTag:
		// Perform an exhaustive search by:
		// * no block or page filters means all blocks and pages match
		// * substitute this trace filter instead rejects everything. therefore it never
		//   quits early due to enough results
		p.tracefilters = append(p.tracefilters, func(s tempofb.Trace) bool {
			return false
		})
		// Skip
		return true, "", ""

	case ErrorTag:
		if v == "true" {
			// Error = true
			return false, StatusCodeTag, strconv.Itoa(int(v1.Status_STATUS_CODE_ERROR))
		}
		// Else fall-through

	case StatusCodeTag:
		// Convert status.code=string into status.code=int
		for statusStr, statusID := range StatusCodeMapping {
			if v == statusStr {
				return false, StatusCodeTag, strconv.Itoa(statusID)
			}
		}
		// Unknown mapping = fall-through
	}

	// No rewrite
	return false, k, v
}

func (p *Pipeline) Matches(e tempofb.Trace) bool {

	for _, f := range p.tracefilters {
		if !f(e) {
			return false
		}
	}

	for _, f := range p.tagfilters {
		if !f(e) {
			return false
		}
	}

	return true
}

func (p *Pipeline) MatchesPage(pg tempofb.Page) bool {
	for _, f := range p.tagfilters {
		if !f(pg) {
			return false
		}
	}

	return true
}

func (p *Pipeline) MatchesBlock(block tempofb.Block) bool {
	for _, f := range p.blockfilters {
		if !f(block) {
			return false
		}
	}

	for _, f := range p.tagfilters {
		if !f(block) {
			return false
		}
	}

	return true
}
