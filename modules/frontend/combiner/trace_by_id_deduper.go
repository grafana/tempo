package combiner

import (
	"encoding/binary"
	"fmt"

	//nolint:all //deprecated

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

const (
	warningTooManySpans = "cannot assign unique span ID, too many spans in the trace"
)

var maxSpanID uint64 = 0xffffffffffffffff

func newDeduper() *spanIDDeduper {
	return &spanIDDeduper{}
}

// This is copied over from Jaeger and modified to work for OpenTelemetry Trace data structure
// https://github.com/jaegertracing/jaeger/blob/12bba8c9b91cf4a29d314934bc08f4a80e43c042/model/adjuster/span_id_deduper.go
type spanIDDeduper struct {
	trace     *tempopb.Trace
	spansByID map[uint64][]*v1.Span
	maxUsedID uint64
}

func (s *spanIDDeduper) dedupe(trace *tempopb.Trace) *tempopb.Trace {
	if trace == nil {
		return nil
	}

	s.trace = trace

	s.groupSpansByID()
	s.dedupeSpanIDs()

	return s.trace
}

// groupSpansByID groups spans with the same ID returning a map id -> []Span
func (s *spanIDDeduper) groupSpansByID() {
	spansByID := make(map[uint64][]*v1.Span)
	for _, batch := range s.trace.Batches {
		for _, ils := range batch.ScopeSpans {
			for _, span := range ils.Spans {
				id := binary.BigEndian.Uint64(span.SpanId)
				if spans, ok := spansByID[id]; ok {
					// TODO maybe return an error if more than 2 spans found
					spansByID[id] = append(spans, span)
				} else {
					spansByID[id] = []*v1.Span{span}
				}
			}
		}
	}
	s.spansByID = spansByID
}

func (s *spanIDDeduper) isSharedWithClientSpan(spanID uint64) bool {
	for _, span := range s.spansByID[spanID] {
		if span.GetKind() == v1.Span_SPAN_KIND_CLIENT {
			return true
		}
	}
	return false
}

func (s *spanIDDeduper) dedupeSpanIDs() {
	oldToNewSpanIDs := make(map[uint64]uint64)
	for _, batch := range s.trace.Batches {
		for _, ils := range batch.ScopeSpans {
			for _, span := range ils.Spans {
				id := binary.BigEndian.Uint64(span.SpanId)
				// only replace span IDs for server-side spans that share the ID with something else
				if span.GetKind() == v1.Span_SPAN_KIND_SERVER && s.isSharedWithClientSpan(id) {
					newID, err := s.makeUniqueSpanID()
					if err != nil {
						// ignore this error condition where we have more than 2^64 unique span IDs
						continue
					}
					oldToNewSpanIDs[id] = newID
					if len(span.ParentSpanId) == 0 {
						span.ParentSpanId = make([]byte, 8)
					}
					binary.BigEndian.PutUint64(span.ParentSpanId, id) // previously shared ID is the new parent
					binary.BigEndian.PutUint64(span.SpanId, newID)
				}
			}
		}
	}
	s.swapParentIDs(oldToNewSpanIDs)
}

// swapParentIDs corrects ParentSpanID of all spans that are children of the server
// spans whose IDs we deduped.
func (s *spanIDDeduper) swapParentIDs(oldToNewSpanIDs map[uint64]uint64) {
	if len(oldToNewSpanIDs) == 0 {
		return
	}
	for _, batch := range s.trace.Batches {
		for _, ils := range batch.ScopeSpans {
			for _, span := range ils.Spans {
				if len(span.GetParentSpanId()) > 0 {
					parentSpanID := binary.BigEndian.Uint64(span.GetParentSpanId())
					if newParentID, ok := oldToNewSpanIDs[parentSpanID]; ok {
						if binary.BigEndian.Uint64(span.SpanId) != newParentID {
							binary.BigEndian.PutUint64(span.ParentSpanId, newParentID)
						}
					}
				}
			}
		}
	}
}

// makeUniqueSpanID returns a new ID that is not used in the trace,
// or an error if such ID cannot be generated, which is unlikely,
// given that the whole space of span IDs is 2^64.
func (s *spanIDDeduper) makeUniqueSpanID() (uint64, error) {
	for id := s.maxUsedID + 1; id < maxSpanID; id++ {
		if _, ok := s.spansByID[id]; !ok {
			s.maxUsedID = id
			return id, nil
		}
	}
	return 0, fmt.Errorf(warningTooManySpans)
}
