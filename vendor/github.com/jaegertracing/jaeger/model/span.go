// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"encoding/gob"
	"io"
	"strconv"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type SamplerType int

const (
	SamplerTypeUnrecognized SamplerType = iota
	SamplerTypeProbabilistic
	SamplerTypeLowerBound
	SamplerTypeRateLimiting
	SamplerTypeConst

	// SampledFlag is the bit set in Flags in order to define a span as a sampled span
	SampledFlag = Flags(1)
	// DebugFlag is the bit set in Flags in order to define a span as a debug span
	DebugFlag = Flags(2)
	// FirehoseFlag is the bit in Flags in order to define a span as a firehose span
	FirehoseFlag = Flags(8)

	keySamplerType  = "sampler.type"
	keySpanKind     = "span.kind"
	keySamplerParam = "sampler.param"
)

// Flags is a bit map of flags for a span
type Flags uint32

var toSpanKind = map[string]trace.SpanKind{
	"client":   trace.SpanKindClient,
	"server":   trace.SpanKindServer,
	"producer": trace.SpanKindProducer,
	"consumer": trace.SpanKindConsumer,
	"internal": trace.SpanKindInternal,
}

var toSamplerType = map[string]SamplerType{
	"unrecognized":  SamplerTypeUnrecognized,
	"probabilistic": SamplerTypeProbabilistic,
	"lowerbound":    SamplerTypeLowerBound,
	"ratelimiting":  SamplerTypeRateLimiting,
	"const":         SamplerTypeConst,
}

func (s SamplerType) String() string {
	switch s {
	case SamplerTypeUnrecognized:
		return "unrecognized"
	case SamplerTypeProbabilistic:
		return "probabilistic"
	case SamplerTypeLowerBound:
		return "lowerbound"
	case SamplerTypeRateLimiting:
		return "ratelimiting"
	case SamplerTypeConst:
		return "const"
	default:
		return ""
	}
}

// Hash implements Hash from Hashable.
func (s *Span) Hash(w io.Writer) (err error) {
	// gob is not the most efficient way, but it ensures we don't miss any fields.
	// See BenchmarkSpanHash in span_test.go
	enc := gob.NewEncoder(w)
	return enc.Encode(s)
}

// HasSpanKind returns true if the span has a `span.kind` tag set to `kind`.
func (s *Span) HasSpanKind(kind trace.SpanKind) bool {
	if tag, ok := KeyValues(s.Tags).FindByKey(keySpanKind); ok {
		return tag.AsString() == kind.String()
	}
	return false
}

// GetSpanKind returns value of `span.kind` tag and whether the tag can be found
func (s *Span) GetSpanKind() (spanKind trace.SpanKind, found bool) {
	if tag, ok := KeyValues(s.Tags).FindByKey(keySpanKind); ok {
		if kind, ok := toSpanKind[tag.AsString()]; ok {
			return kind, true
		}
	}
	return trace.SpanKindUnspecified, false
}

// GetSamplerType returns the sampler type for span
func (s *Span) GetSamplerType() SamplerType {
	// There's no corresponding opentelemetry tag label corresponding to sampler.type
	if tag, ok := KeyValues(s.Tags).FindByKey(keySamplerType); ok {
		if s, ok := toSamplerType[tag.VStr]; ok {
			return s
		}
	}
	return SamplerTypeUnrecognized
}

// IsRPCClient returns true if the span represents a client side of an RPC,
// as indicated by the `span.kind` tag set to `client`.
func (s *Span) IsRPCClient() bool {
	return s.HasSpanKind(trace.SpanKindClient)
}

// IsRPCServer returns true if the span represents a server side of an RPC,
// as indicated by the `span.kind` tag set to `server`.
func (s *Span) IsRPCServer() bool {
	return s.HasSpanKind(trace.SpanKindServer)
}

// NormalizeTimestamps changes all timestamps in this span to UTC.
func (s *Span) NormalizeTimestamps() {
	s.StartTime = s.StartTime.UTC()
	for i := range s.Logs {
		s.Logs[i].Timestamp = s.Logs[i].Timestamp.UTC()
	}
}

// ParentSpanID returns ID of a parent span if it exists.
// It searches for the first child-of or follows-from reference pointing to the same trace ID.
func (s *Span) ParentSpanID() SpanID {
	var followsFromRef *SpanRef
	for i := range s.References {
		ref := &s.References[i]
		if ref.TraceID != s.TraceID {
			continue
		}
		if ref.RefType == ChildOf {
			return ref.SpanID
		}
		if followsFromRef == nil && ref.RefType == FollowsFrom {
			followsFromRef = ref
		}
	}
	if followsFromRef != nil {
		return followsFromRef.SpanID
	}
	return SpanID(0)
}

// ReplaceParentID replaces span ID in the parent span reference.
// See also ParentSpanID.
func (s *Span) ReplaceParentID(newParentID SpanID) {
	oldParentID := s.ParentSpanID()
	for i := range s.References {
		if s.References[i].SpanID == oldParentID && s.References[i].TraceID == s.TraceID {
			s.References[i].SpanID = newParentID
			return
		}
	}
	s.References = MaybeAddParentSpanID(s.TraceID, newParentID, s.References)
}

// GetSamplerParams returns the sampler.type and sampler.param value if they are valid.
func (s *Span) GetSamplerParams(logger *zap.Logger) (SamplerType, float64) {
	samplerType := s.GetSamplerType()
	if samplerType == SamplerTypeUnrecognized {
		return SamplerTypeUnrecognized, 0
	}
	tag, ok := KeyValues(s.Tags).FindByKey(keySamplerParam)
	if !ok {
		return SamplerTypeUnrecognized, 0
	}
	samplerParam, err := samplerParamToFloat(tag)
	if err != nil {
		logger.
			With(zap.String("traceID", s.TraceID.String())).
			With(zap.String("spanID", s.SpanID.String())).
			Warn("sampler.param tag is not a number", zap.Any("tag", tag))
		return SamplerTypeUnrecognized, 0
	}
	return samplerType, samplerParam
}

// ------- Flags -------

// SetSampled sets the Flags as sampled
func (f *Flags) SetSampled() {
	f.setFlags(SampledFlag)
}

// SetDebug set the Flags as sampled
func (f *Flags) SetDebug() {
	f.setFlags(DebugFlag)
}

// SetFirehose set the Flags as firehose enabled
func (f *Flags) SetFirehose() {
	f.setFlags(FirehoseFlag)
}

func (f *Flags) setFlags(bit Flags) {
	*f |= bit
}

// IsSampled returns true if the Flags denote sampling
func (f Flags) IsSampled() bool {
	return f.checkFlags(SampledFlag)
}

// IsDebug returns true if the Flags denote debugging
// Debugging can be useful in testing tracing availability or correctness
func (f Flags) IsDebug() bool {
	return f.checkFlags(DebugFlag)
}

// IsFirehoseEnabled returns true if firehose is enabled
// Firehose is used to decide whether to index a span or not
func (f Flags) IsFirehoseEnabled() bool {
	return f.checkFlags(FirehoseFlag)
}

func (f Flags) checkFlags(bit Flags) bool {
	return f&bit == bit
}

func samplerParamToFloat(samplerParamTag KeyValue) (float64, error) {
	// The param could be represented as a string, an int, or a float
	switch samplerParamTag.VType {
	case Float64Type:
		return samplerParamTag.Float64(), nil
	case Int64Type:
		return float64(samplerParamTag.Int64()), nil
	default:
		return strconv.ParseFloat(samplerParamTag.AsString(), 64)
	}
}
