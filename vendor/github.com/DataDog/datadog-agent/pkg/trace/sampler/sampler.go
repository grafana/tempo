// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package sampler contains all the logic of the agent-side trace sampling
package sampler

import (
	"math"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/traceutil"
)

const (
	// KeySamplingRateGlobal is a metric key holding the global sampling rate.
	KeySamplingRateGlobal = "_sample_rate"

	// KeySamplingRateClient is a metric key holding the client-set sampling rate for APM events.
	KeySamplingRateClient = "_dd1.sr.rcusr"

	// KeySamplingRatePreSampler is a metric key holding the API rate limiter's rate for APM events.
	KeySamplingRatePreSampler = "_dd1.sr.rapre"

	// KeySamplingRateEventExtraction is the key of the metric storing the event extraction rate on an APM event.
	KeySamplingRateEventExtraction = "_dd1.sr.eausr"

	// KeySamplingRateMaxEPSSampler is the key of the metric storing the max eps sampler rate on an APM event.
	KeySamplingRateMaxEPSSampler = "_dd1.sr.eamax"

	// KeyErrorType is the key of the error type in the meta map
	KeyErrorType = "error.type"

	// KeyAnalyzedSpans is the metric key which specifies if a span is analyzed.
	KeyAnalyzedSpans = "_dd.analyzed"

	// KeyHTTPStatusCode is the key of the http status code in the meta map
	KeyHTTPStatusCode = "http.status_code"

	// KeySpanSamplingMechanism is the metric key holding a span sampling rule that a span was kept on.
	KeySpanSamplingMechanism = "_dd.span_sampling.mechanism"
)

// SamplingPriority is the type encoding a priority sampling decision.
type SamplingPriority int8

const (
	// PriorityNone is the value for SamplingPriority when no priority sampling decision could be found.
	PriorityNone SamplingPriority = math.MinInt8

	// PriorityUserDrop is the value set by a user to explicitly drop a trace.
	PriorityUserDrop SamplingPriority = -1

	// PriorityAutoDrop is the value set by a tracer to suggest dropping a trace.
	PriorityAutoDrop SamplingPriority = 0

	// PriorityAutoKeep is the value set by a tracer to suggest keeping a trace.
	PriorityAutoKeep SamplingPriority = 1

	// PriorityUserKeep is the value set by a user to explicitly keep a trace.
	PriorityUserKeep SamplingPriority = 2

	// 2^64 - 1
	maxTraceID      = ^uint64(0)
	maxTraceIDFloat = float64(maxTraceID)
	// Good number for Knuth hashing (large, prime, fit in int64 for languages without uint64)
	samplerHasher = uint64(1111111111111111111)
)

// SampleByRate returns whether to keep a trace, based on its ID and a sampling rate.
// This assumes that trace IDs are nearly uniformly distributed.
func SampleByRate(traceID uint64, rate float64) bool {
	if rate < 1 {
		return traceID*samplerHasher < uint64(rate*maxTraceIDFloat)
	}
	return true
}

// GetSamplingPriority returns the value of the sampling priority metric set on this span and a boolean indicating if
// such a metric was actually found or not.
func GetSamplingPriority(t *pb.TraceChunk) (SamplingPriority, bool) {
	if t.Priority == int32(PriorityNone) {
		return 0, false
	}
	return SamplingPriority(t.Priority), true
}

// GetGlobalRate gets the cumulative sample rate of the trace to which this span belongs to.
func GetGlobalRate(s *pb.Span) float64 {
	return getMetricDefault(s, KeySamplingRateGlobal, 1.0)
}

// GetClientRate gets the rate at which the trace this span belongs to was sampled by the tracer.
// NOTE: This defaults to 1 if no rate is stored.
func GetClientRate(s *pb.Span) float64 {
	return getMetricDefault(s, KeySamplingRateClient, 1.0)
}

// SetClientRate sets the rate at which the trace this span belongs to was sampled by the tracer.
func SetClientRate(s *pb.Span, rate float64) {
	if rate < 1 {
		setMetric(s, KeySamplingRateClient, rate)
	} else {
		// We assume missing value is 1 to save bandwidth (check getter).
		delete(s.Metrics, KeySamplingRateClient)
	}
}

// GetPreSampleRate returns the rate at which the trace this span belongs to was sampled by the agent's presampler.
// NOTE: This defaults to 1 if no rate is stored.
func GetPreSampleRate(s *pb.Span) float64 {
	return getMetricDefault(s, KeySamplingRatePreSampler, 1.0)
}

// SetPreSampleRate sets the rate at which the trace this span belongs to was sampled by the agent's presampler.
func SetPreSampleRate(s *pb.Span, rate float64) {
	if rate < 1 {
		setMetric(s, KeySamplingRatePreSampler, rate)
	} else {
		// We assume missing value is 1 to save bandwidth (check getter).
		delete(s.Metrics, KeySamplingRatePreSampler)
	}
}

// GetEventExtractionRate gets the rate at which the trace from which we extracted this event was sampled at the tracer.
// This defaults to 1 if no rate is stored.
func GetEventExtractionRate(s *pb.Span) float64 {
	return getMetricDefault(s, KeySamplingRateEventExtraction, 1.0)
}

// SetEventExtractionRate sets the rate at which the trace from which we extracted this event was sampled at the tracer.
func SetEventExtractionRate(s *pb.Span, rate float64) {
	if rate < 1 {
		setMetric(s, KeySamplingRateEventExtraction, rate)
	} else {
		// reduce bandwidth, default is assumed 1.0 in backend
		delete(s.Metrics, KeySamplingRateEventExtraction)
	}
}

// GetMaxEPSRate gets the rate at which this event was sampled by the max eps event sampler.
func GetMaxEPSRate(s *pb.Span) float64 {
	return getMetricDefault(s, KeySamplingRateMaxEPSSampler, 1.0)
}

// SetMaxEPSRate sets the rate at which this event was sampled by the max eps event sampler.
func SetMaxEPSRate(s *pb.Span, rate float64) {
	if rate < 1 {
		setMetric(s, KeySamplingRateMaxEPSSampler, rate)
	} else {
		// reduce bandwidth, default is assumed 1.0 in backend
		delete(s.Metrics, KeySamplingRateMaxEPSSampler)
	}
}

// SetAnalyzedSpan marks a span analyzed
func SetAnalyzedSpan(s *pb.Span) {
	setMetric(s, KeyAnalyzedSpans, 1)
}

// IsAnalyzedSpan checks if a span is analyzed
func IsAnalyzedSpan(s *pb.Span) bool {
	v, _ := getMetric(s, KeyAnalyzedSpans)
	return v == 1
}

func weightRoot(s *pb.Span) float32 {
	if s == nil {
		return 1
	}
	clientRate, ok := s.Metrics[KeySamplingRateGlobal]
	if !ok || clientRate <= 0.0 || clientRate > 1.0 {
		clientRate = 1
	}
	preSamplerRate, ok := s.Metrics[KeySamplingRatePreSampler]
	if !ok || preSamplerRate <= 0.0 || preSamplerRate > 1.0 {
		preSamplerRate = 1
	}
	return float32(1.0 / (preSamplerRate * clientRate))
}

func getMetric(s *pb.Span, k string) (float64, bool) {
	if s.Metrics == nil {
		return 0, false
	}
	val, ok := s.Metrics[k]
	return val, ok
}

// getMetricDefault gets a value in the span Metrics map or default if no value is stored there.
func getMetricDefault(s *pb.Span, k string, def float64) float64 {
	if val, ok := getMetric(s, k); ok {
		return val
	}
	return def
}

// setMetric sets a value in the span Metrics map.
func setMetric(s *pb.Span, key string, val float64) {
	if s.Metrics == nil {
		s.Metrics = make(map[string]float64)
	}
	s.Metrics[key] = val
}

// ApplySpanSampling searches chunk for spans that have a span sampling tag set.
// If it finds such spans, then it replaces chunk's spans with only those spans,
// and sets the chunk's sampling priority to "user keep." Tracers that wish to
// keep certain spans even when the trace is dropped will set the appropriate
// tags on the spans to be kept.
// ApplySpanSampling returns whether any changes were actually made.
// Do not call ApplySpanSampling on a chunk that the other samplers have
// decided to keep. Doing so might wrongfully remove spans from a kept trace.
func ApplySpanSampling(chunk *pb.TraceChunk) bool {
	var sampledSpans []*pb.Span
	for _, span := range chunk.Spans {
		if _, ok := traceutil.GetMetric(span, KeySpanSamplingMechanism); ok {
			// Keep only those spans that have a span sampling tag.
			sampledSpans = append(sampledSpans, span)
		}
	}
	if sampledSpans == nil {
		// No span sampling tags → no span sampling.
		return false
	}

	chunk.Spans = sampledSpans
	chunk.Priority = int32(PriorityUserKeep)
	chunk.DroppedTrace = false
	return true
}
