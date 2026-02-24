package livestore

import (
	"bytes"
	"context"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// protoSearcher implements common.Searcher and common.Finder over a snapshot
// of proto traces (from liveTraces or pendingTraces). It is immutable for the
// duration of a query.
type protoSearcher struct {
	traces []*traceSnapshot
	meta   *backend.BlockMeta
}

// traceSnapshot is an immutable view of a single trace for querying.
type traceSnapshot struct {
	id      []byte
	batches []*trace_v1.ResourceSpans
}

var (
	_ common.Searcher = (*protoSearcher)(nil)
	_ common.Finder   = (*protoSearcher)(nil)
)

// newProtoSearcherFromLiveTraces creates a searcher from a snapshot of live traces.
func newProtoSearcherFromLiveTraces(traces map[uint64]*traceSnapshot) *protoSearcher {
	flat := make([]*traceSnapshot, 0, len(traces))
	for _, t := range traces {
		flat = append(flat, t)
	}
	return newProtoSearcher(flat)
}

// newProtoSearcherFromPending creates a searcher from a snapshot of pending traces.
func newProtoSearcherFromPending(pending []*pendingTrace) *protoSearcher {
	traces := make([]*traceSnapshot, 0, len(pending))
	for _, p := range pending {
		traces = append(traces, &traceSnapshot{
			id:      p.ID,
			batches: p.Batches,
		})
	}
	return newProtoSearcher(traces)
}

func newProtoSearcher(traces []*traceSnapshot) *protoSearcher {
	var minStart, maxEnd time.Time

	for _, t := range traces {
		start, end := traceTimeBounds(t.batches)
		startTime := time.Unix(0, int64(start))
		endTime := time.Unix(0, int64(end))

		if minStart.IsZero() || startTime.Before(minStart) {
			minStart = startTime
		}
		if endTime.After(maxEnd) {
			maxEnd = endTime
		}
	}

	if minStart.IsZero() {
		minStart = time.Now()
	}
	if maxEnd.IsZero() {
		maxEnd = time.Now()
	}

	meta := &backend.BlockMeta{
		BlockID:   backend.NewUUID(),
		StartTime: minStart,
		EndTime:   maxEnd,
	}

	return &protoSearcher{
		traces: traces,
		meta:   meta,
	}
}

func (s *protoSearcher) BlockMeta() *backend.BlockMeta {
	return s.meta
}

func (s *protoSearcher) FindTraceByID(_ context.Context, id common.ID, _ common.SearchOptions) (*tempopb.TraceByIDResponse, error) {
	for _, t := range s.traces {
		if bytes.Equal(t.id, id) {
			return &tempopb.TraceByIDResponse{
				Trace: &tempopb.Trace{
					ResourceSpans: t.batches,
				},
			}, nil
		}
		// Also try matching by trace ID embedded in spans
		if traceMatchesID(t.batches, id) {
			return &tempopb.TraceByIDResponse{
				Trace: &tempopb.Trace{
					ResourceSpans: t.batches,
				},
			}, nil
		}
	}
	return nil, nil
}

func (s *protoSearcher) Search(_ context.Context, req *tempopb.SearchRequest, _ common.SearchOptions) (*tempopb.SearchResponse, error) {
	// Proto searcher doesn't handle non-TraceQL text search.
	// All search goes through Fetch -> engine.ExecuteSearch in iterateBlocks.
	return &tempopb.SearchResponse{}, nil
}

func (s *protoSearcher) Fetch(_ context.Context, req traceql.FetchSpansRequest, _ common.SearchOptions) (traceql.FetchSpansResponse, error) {
	wantNestedSet := needsNestedSet(req)
	var totalBytes uint64

	var spansets []*traceql.Spanset
	for _, t := range s.traces {
		spans := extractSpansFromTrace(t.batches)
		if len(spans) == 0 {
			continue
		}

		if wantNestedSet {
			buildNestedSet(spans)
		}

		// Estimate bytes for this trace
		for _, batch := range t.batches {
			totalBytes += uint64(batch.Size())
		}

		// Build traceql.Spans and evaluate conditions
		traceqlSpans := make([]traceql.Span, 0, len(spans))
		for _, ps := range spans {
			if matchesConditions(ps, req) {
				traceqlSpans = append(traceqlSpans, ps)
			}
		}

		if len(traceqlSpans) == 0 {
			continue
		}

		// Build spanset metadata
		traceID := traceIDFromSpans(t.batches)
		startNanos, endNanos := traceTimeBounds(t.batches)
		var durationNanos uint64
		if endNanos > startNanos {
			durationNanos = endNanos - startNanos
		}

		rootSpan, rootService := findRootSpanAndService(t.batches)

		ss := &traceql.Spanset{
			TraceID:            traceID,
			RootSpanName:       rootSpan,
			RootServiceName:    rootService,
			StartTimeUnixNanos: startNanos,
			DurationNanos:      durationNanos,
			Spans:              traceqlSpans,
			ServiceStats:       computeServiceStats(t.batches),
		}

		spansets = append(spansets, ss)
	}

	bytesVal := totalBytes
	return traceql.FetchSpansResponse{
		Results: newProtoSpansetIterator(spansets),
		Bytes:   func() uint64 { return bytesVal },
	}, nil
}

func (s *protoSearcher) SearchTags(_ context.Context, scope traceql.AttributeScope, cb common.TagsCallback, mcb common.MetricsCallback, _ common.SearchOptions) error {
	seen := make(map[string]struct{})
	var totalBytes uint64

	for _, t := range s.traces {
		for _, rs := range t.batches {
			totalBytes += uint64(rs.Size())

			if scope == traceql.AttributeScopeResource || scope == traceql.AttributeScopeNone {
				if rs.Resource != nil {
					for _, kv := range rs.Resource.Attributes {
						if _, ok := seen[kv.Key]; !ok {
							seen[kv.Key] = struct{}{}
							cb(kv.Key, traceql.AttributeScopeResource)
						}
					}
				}
			}

			for _, ss := range rs.ScopeSpans {
				if scope == traceql.AttributeScopeSpan || scope == traceql.AttributeScopeNone {
					for _, span := range ss.Spans {
						for _, kv := range span.Attributes {
							key := kv.Key
							if _, ok := seen[key]; !ok {
								seen[key] = struct{}{}
								cb(key, traceql.AttributeScopeSpan)
							}
						}
					}
				}

				if scope == traceql.AttributeScopeEvent || scope == traceql.AttributeScopeNone {
					for _, span := range ss.Spans {
						for _, event := range span.Events {
							for _, kv := range event.Attributes {
								key := kv.Key
								if _, ok := seen[key]; !ok {
									seen[key] = struct{}{}
									cb(key, traceql.AttributeScopeEvent)
								}
							}
						}
					}
				}

				if scope == traceql.AttributeScopeLink || scope == traceql.AttributeScopeNone {
					for _, span := range ss.Spans {
						for _, link := range span.Links {
							for _, kv := range link.Attributes {
								key := kv.Key
								if _, ok := seen[key]; !ok {
									seen[key] = struct{}{}
									cb(key, traceql.AttributeScopeLink)
								}
							}
						}
					}
				}

				if scope == traceql.AttributeScopeInstrumentation || scope == traceql.AttributeScopeNone {
					if ss.Scope != nil {
						for _, kv := range ss.Scope.Attributes {
							key := kv.Key
							if _, ok := seen[key]; !ok {
								seen[key] = struct{}{}
								cb(key, traceql.AttributeScopeInstrumentation)
							}
						}
					}
				}
			}
		}
	}

	if mcb != nil {
		mcb(totalBytes)
	}
	return nil
}

// SearchTagValues searches for tag values matching the given unscoped tag name.
// Matches parquet block behavior: for unscoped tags, only resource, span, and
// instrumentation attributes are searched (not events or links).
func (s *protoSearcher) SearchTagValues(_ context.Context, tag string, cb common.TagValuesCallback, mcb common.MetricsCallback, _ common.SearchOptions) error {
	var totalBytes uint64

	for _, t := range s.traces {
		for _, rs := range t.batches {
			totalBytes += uint64(rs.Size())

			if rs.Resource != nil {
				for _, kv := range rs.Resource.Attributes {
					if kv.Key == tag {
						if str := keyValueToString(kv); str != "" {
							cb(str)
						}
					}
				}
			}

			for _, ss := range rs.ScopeSpans {
				for _, span := range ss.Spans {
					for _, kv := range span.Attributes {
						if kv.Key == tag {
							if str := keyValueToString(kv); str != "" {
								cb(str)
							}
						}
					}
				}
				if ss.Scope != nil {
					for _, kv := range ss.Scope.Attributes {
						if kv.Key == tag {
							if str := keyValueToString(kv); str != "" {
								cb(str)
							}
						}
					}
				}
			}
		}
	}

	if mcb != nil {
		mcb(totalBytes)
	}
	return nil
}

func (s *protoSearcher) SearchTagValuesV2(_ context.Context, tag traceql.Attribute, cb common.TagValuesCallbackV2, mcb common.MetricsCallback, _ common.SearchOptions) error {
	var totalBytes uint64

	for _, t := range s.traces {
		spans := extractSpansFromTrace(t.batches)
		for _, batch := range t.batches {
			totalBytes += uint64(batch.Size())
		}

		for _, ps := range spans {
			val, ok := ps.AttributeFor(tag)
			if ok && val.Type != traceql.TypeNil {
				if cb(val) {
					goto done
				}
			}
		}
	}

done:
	if mcb != nil {
		mcb(totalBytes)
	}
	return nil
}

func (s *protoSearcher) FetchTagValues(_ context.Context, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback, mcb common.MetricsCallback, _ common.SearchOptions) error {
	var totalBytes uint64

	for _, t := range s.traces {
		spans := extractSpansFromTrace(t.batches)
		for _, batch := range t.batches {
			totalBytes += uint64(batch.Size())
		}

		for _, ps := range spans {
			// Check if span matches the filter conditions
			if !matchesFetchConditions(ps, req.Conditions) {
				continue
			}

			val, ok := ps.AttributeFor(req.TagName)
			if ok && val.Type != traceql.TypeNil {
				if cb(val) {
					goto done
				}
			}
		}
	}

done:
	if mcb != nil {
		mcb(totalBytes)
	}
	return nil
}

func (s *protoSearcher) FetchTagNames(_ context.Context, req traceql.FetchTagsRequest, cb traceql.FetchTagsCallback, mcb common.MetricsCallback, _ common.SearchOptions) error {
	seen := make(map[string]struct{})
	var totalBytes uint64

	for _, t := range s.traces {
		spans := extractSpansFromTrace(t.batches)
		for _, batch := range t.batches {
			totalBytes += uint64(batch.Size())
		}

		for _, ps := range spans {
			// Check if span matches the filter conditions
			if !matchesFetchConditions(ps, req.Conditions) {
				continue
			}

			ps.AllAttributesFunc(func(a traceql.Attribute, _ traceql.Static) {
				if a.Intrinsic != traceql.IntrinsicNone {
					return
				}
				if req.Scope != traceql.AttributeScopeNone && a.Scope != req.Scope {
					return
				}
				key := a.Scope.String() + ":" + a.Name
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					cb(a.Name, a.Scope)
				}
			})
		}
	}

	if mcb != nil {
		mcb(totalBytes)
	}
	return nil
}

// matchesConditions checks if a span satisfies all conditions (if AllConditions)
// or any condition in the request.
func matchesConditions(s *protoSpan, req traceql.FetchSpansRequest) bool {
	if len(req.Conditions) == 0 {
		return true
	}

	for _, c := range req.Conditions {
		_, ok := s.AttributeFor(c.Attribute)
		if ok {
			if !req.AllConditions {
				return true
			}
		} else if req.AllConditions {
			return false
		}
	}

	return req.AllConditions
}

// matchesFetchConditions checks if a span matches a set of conditions (for tag queries).
func matchesFetchConditions(s *protoSpan, conditions []traceql.Condition) bool {
	if len(conditions) == 0 {
		return true
	}
	for _, c := range conditions {
		_, ok := s.AttributeFor(c.Attribute)
		if !ok {
			return false
		}
	}
	return true
}

// findRootSpanAndService finds the root span name and service name for a trace.
func findRootSpanAndService(batches []*trace_v1.ResourceSpans) (rootSpanName, rootServiceName string) {
	for _, rs := range batches {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if len(span.ParentSpanId) == 0 {
					rootSpanName = span.Name
					if rs.Resource != nil {
						for _, kv := range rs.Resource.Attributes {
							if kv.Key == "service.name" {
								if sv, ok := kv.Value.Value.(*v1.AnyValue_StringValue); ok {
									rootServiceName = sv.StringValue
								}
							}
						}
					}
					return
				}
			}
		}
	}
	return
}

// computeServiceStats computes service stats for a trace (span counts per service).
func computeServiceStats(batches []*trace_v1.ResourceSpans) map[string]traceql.ServiceStats {
	stats := make(map[string]traceql.ServiceStats)
	for _, rs := range batches {
		var serviceName string
		if rs.Resource != nil {
			for _, kv := range rs.Resource.Attributes {
				if kv.Key == "service.name" {
					if sv, ok := kv.Value.Value.(*v1.AnyValue_StringValue); ok {
						serviceName = sv.StringValue
					}
				}
			}
		}
		for _, ss := range rs.ScopeSpans {
			spanCount := uint32(len(ss.Spans))
			var errorCount uint32
			for _, span := range ss.Spans {
				if span.Status != nil && span.Status.Code == trace_v1.Status_STATUS_CODE_ERROR {
					errorCount++
				}
			}
			existing := stats[serviceName]
			existing.SpanCount += spanCount
			existing.ErrorCount += errorCount
			stats[serviceName] = existing
		}
	}
	return stats
}

// keyValueToString converts a KeyValue's value to a string representation.
func keyValueToString(kv *v1.KeyValue) string {
	if kv.Value == nil {
		return ""
	}
	switch v := kv.Value.Value.(type) {
	case *v1.AnyValue_StringValue:
		return v.StringValue
	case *v1.AnyValue_IntValue:
		return ""
	case *v1.AnyValue_DoubleValue:
		return ""
	case *v1.AnyValue_BoolValue:
		return ""
	default:
		return ""
	}
}

// protoSpansetIterator implements traceql.SpansetIterator.
type protoSpansetIterator struct {
	spansets []*traceql.Spanset
	idx      int
}

var _ traceql.SpansetIterator = (*protoSpansetIterator)(nil)

func newProtoSpansetIterator(spansets []*traceql.Spanset) *protoSpansetIterator {
	return &protoSpansetIterator{spansets: spansets}
}

func (it *protoSpansetIterator) Next(_ context.Context) (*traceql.Spanset, error) {
	if it.idx >= len(it.spansets) {
		return nil, nil
	}
	ss := it.spansets[it.idx]
	it.idx++
	return ss, nil
}

func (it *protoSpansetIterator) Close() {
	it.spansets = nil
}
