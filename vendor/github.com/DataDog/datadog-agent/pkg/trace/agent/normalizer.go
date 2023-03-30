// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package agent

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/trace/traceutil"
)

const (
	// MaxTypeLen the maximum length a span type can have
	MaxTypeLen = 100
	// tagOrigin specifies the origin of the trace.
	// DEPRECATED: Origin is now specified as a TraceChunk field.
	tagOrigin = "_dd.origin"
	// tagSamplingPriority specifies the sampling priority of the trace.
	// DEPRECATED: Priority is now specified as a TraceChunk field.
	tagSamplingPriority = "_sampling_priority_v1"
)

var (
	// Year2000NanosecTS is an arbitrary cutoff to spot weird-looking values
	Year2000NanosecTS = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC).UnixNano()
)

// normalize makes sure a Span is properly initialized and encloses the minimum required info, returning error if it
// is invalid beyond repair
func (a *Agent) normalize(ts *info.TagStats, s *pb.Span) error {
	if s.TraceID == 0 {
		ts.TracesDropped.TraceIDZero.Inc()
		return fmt.Errorf("TraceID is zero (reason:trace_id_zero): %s", s)
	}
	if s.SpanID == 0 {
		ts.TracesDropped.SpanIDZero.Inc()
		return fmt.Errorf("SpanID is zero (reason:span_id_zero): %s", s)
	}
	svc, err := traceutil.NormalizeService(s.Service, ts.Lang)
	switch err {
	case traceutil.ErrEmpty:
		ts.SpansMalformed.ServiceEmpty.Inc()
		log.Debugf("Fixing malformed trace. Service is empty (reason:service_empty), setting span.service=%s: %s", s.Service, s)
	case traceutil.ErrTooLong:
		ts.SpansMalformed.ServiceTruncate.Inc()
		log.Debugf("Fixing malformed trace. Service is too long (reason:service_truncate), truncating span.service to length=%d: %s", traceutil.MaxServiceLen, s)
	case traceutil.ErrInvalid:
		ts.SpansMalformed.ServiceInvalid.Inc()
		log.Debugf("Fixing malformed trace. Service is invalid (reason:service_invalid), replacing invalid span.service=%s with fallback span.service=%s: %s", s.Service, svc, s)
	}
	s.Service = svc

	if a.conf.HasFeature("component2name") {
		// This feature flag determines the component tag to become the span name.
		//
		// It works around the incompatibility between Opentracing and Datadog where the
		// Opentracing operation name is many times invalid as a Datadog operation name (e.g. "/")
		// and in Datadog terms it's the resource. Here, we aim to make the component the
		// operation name to provide a better product experience.
		if v, ok := s.Meta["component"]; ok {
			s.Name = v
		}
	}
	s.Name, err = traceutil.NormalizeName(s.Name)
	switch err {
	case traceutil.ErrEmpty:
		ts.SpansMalformed.SpanNameEmpty.Inc()
		log.Debugf("Fixing malformed trace. Name is empty (reason:span_name_empty), setting span.name=%s: %s", s.Name, s)
	case traceutil.ErrTooLong:
		ts.SpansMalformed.SpanNameTruncate.Inc()
		log.Debugf("Fixing malformed trace. Name is too long (reason:span_name_truncate), truncating span.name to length=%d: %s", traceutil.MaxServiceLen, s)
	case traceutil.ErrInvalid:
		ts.SpansMalformed.SpanNameInvalid.Inc()
		log.Debugf("Fixing malformed trace. Name is invalid (reason:span_name_invalid), setting span.name=%s: %s", s.Name, s)
	}

	if s.Resource == "" {
		ts.SpansMalformed.ResourceEmpty.Inc()
		log.Debugf("Fixing malformed trace. Resource is empty (reason:resource_empty), setting span.resource=%s: %s", s.Name, s)
		s.Resource = s.Name
	}

	// ParentID, TraceID and SpanID set in the client could be the same
	// Supporting the ParentID == TraceID == SpanID for the root span, is compliant
	// with the Zipkin implementation. Furthermore, as described in the PR
	// https://github.com/openzipkin/zipkin/pull/851 the constraint that the
	// root span's ``trace id = span id`` has been removed
	if s.ParentID == s.TraceID && s.ParentID == s.SpanID {
		s.ParentID = 0
		log.Debugf("span.normalize: `ParentID`, `TraceID` and `SpanID` are the same; `ParentID` set to 0: %d", s.TraceID)
	}

	// Start & Duration as nanoseconds timestamps
	// if s.Start is very little, less than year 2000 probably a unit issue so discard
	// (or it is "le bug de l'an 2000")
	if s.Duration < 0 {
		ts.SpansMalformed.InvalidDuration.Inc()
		log.Debugf("Fixing malformed trace. Duration is invalid (reason:invalid_duration), setting span.duration=0: %s", s)
		s.Duration = 0
	}
	if s.Duration > math.MaxInt64-s.Start {
		ts.SpansMalformed.InvalidDuration.Inc()
		log.Debugf("Fixing malformed trace. Duration is too large and causes overflow (reason:invalid_duration), setting span.duration=0: %s", s)
		s.Duration = 0
	}
	if s.Start < Year2000NanosecTS {
		ts.SpansMalformed.InvalidStartDate.Inc()
		log.Debugf("Fixing malformed trace. Start date is invalid (reason:invalid_start_date), setting span.start=time.now(): %s", s)
		now := time.Now().UnixNano()
		s.Start = now - s.Duration
		if s.Start < 0 {
			s.Start = now
		}
	}

	if len(s.Type) > MaxTypeLen {
		ts.SpansMalformed.TypeTruncate.Inc()
		log.Debugf("Fixing malformed trace. Type is too long (reason:type_truncate), truncating span.type to length=%d: %s", MaxTypeLen, s)
		s.Type = traceutil.TruncateUTF8(s.Type, MaxTypeLen)
	}
	if env, ok := s.Meta["env"]; ok {
		s.Meta["env"] = traceutil.NormalizeTag(env)
	}
	if sc, ok := s.Meta["http.status_code"]; ok {
		if !isValidStatusCode(sc) {
			ts.SpansMalformed.InvalidHTTPStatusCode.Inc()
			log.Debugf("Fixing malformed trace. HTTP status code is invalid (reason:invalid_http_status_code), dropping invalid http.status_code=%s: %s", sc, s)
			delete(s.Meta, "http.status_code")
		}
	}
	return nil
}

// normalizeChunk takes a trace chunk and
// * populates Origin field if it wasn't populated
// * populates Priority field if it wasn't populated
func normalizeChunk(chunk *pb.TraceChunk, root *pb.Span) {
	// check if priority is already populated
	if chunk.Priority == int32(sampler.PriorityNone) {
		// Older tracers set sampling priority in the root span.
		if p, ok := root.Metrics[tagSamplingPriority]; ok {
			chunk.Priority = int32(p)
		} else {
			for _, s := range chunk.Spans {
				if p, ok := s.Metrics[tagSamplingPriority]; ok {
					chunk.Priority = int32(p)
					break
				}
			}
		}
	}
	if chunk.Origin == "" && root.Meta != nil {
		// Older tracers set origin in the root span.
		chunk.Origin = root.Meta[tagOrigin]
	}
}

// normalizeTrace takes a trace and
// * rejects the trace if there is a trace ID discrepancy between 2 spans
// * rejects the trace if two spans have the same span_id
// * rejects empty traces
// * rejects traces where at least one span cannot be normalized
// * return the normalized trace and an error:
//   - nil if the trace can be accepted
//   - a reason tag explaining the reason the traces failed normalization
func (a *Agent) normalizeTrace(ts *info.TagStats, t pb.Trace) error {
	if len(t) == 0 {
		ts.TracesDropped.EmptyTrace.Inc()
		return errors.New("trace is empty (reason:empty_trace)")
	}

	spanIDs := make(map[uint64]struct{})
	firstSpan := t[0]

	for _, span := range t {
		if span == nil {
			continue
		}
		if firstSpan == nil {
			firstSpan = span
		}
		if span.TraceID != firstSpan.TraceID {
			ts.TracesDropped.ForeignSpan.Inc()
			return fmt.Errorf("trace has foreign span (reason:foreign_span): %s", span)
		}
		if err := a.normalize(ts, span); err != nil {
			return err
		}
		if _, ok := spanIDs[span.SpanID]; ok {
			ts.SpansMalformed.DuplicateSpanID.Inc()
			log.Debugf("Found malformed trace with duplicate span ID (reason:duplicate_span_id): %s", span)
		}
		spanIDs[span.SpanID] = struct{}{}
	}

	return nil
}

func (a *Agent) normalizeStatsGroup(b *pb.ClientGroupedStats, lang string) {
	b.Name, _ = traceutil.NormalizeName(b.Name)
	b.Service, _ = traceutil.NormalizeService(b.Service, lang)
	if b.Resource == "" {
		b.Resource = b.Name
	}
	b.Resource, _ = a.TruncateResource(b.Resource)
}

func isValidStatusCode(sc string) bool {
	if code, err := strconv.ParseUint(sc, 10, 64); err == nil {
		return 100 <= code && code < 600
	}
	return false
}
